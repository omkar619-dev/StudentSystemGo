// Package photo wraps S3 + CloudFront for entity profile photos.
//
// Two flows:
//
//  1. UPLOAD — client gets a presigned S3 PUT URL from us, then PUTs the
//     image directly to S3. Bypasses our app server (no upload bandwidth).
//  2. READ   — client gets a CloudFront signed URL from us, then GETs the
//     image via CloudFront's CDN edge.
//
// The bucket is fully private. Every read needs a signed URL. Every write
// needs a presigned URL. Nobody hits S3 anonymously.
package photo

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/cloudfront/sign"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Config holds per-deployment values from env vars.
//
// Why a struct (instead of just reading env in each function): testability
// and explicitness. With a Config struct, the Init function takes one obvious
// argument; tests can pass fake values without polluting os.Setenv.
type Config struct {
	Bucket          string // e.g., "studentsystemgo-uploads-a7k3m2"
	Region          string // e.g., "ap-south-1"
	CFDomain        string // e.g., "d2pv7o3f0grsso.cloudfront.net" (no scheme)
	CFKeyPairID     string // e.g., "K3LVUK5HGD8IW2" — the CloudFront public key ID
	CFPrivateKeyPth string // path to PEM-encoded RSA private key on disk
}

// Module-level state.
//
// We keep ONE S3 client and ONE presign client for the whole app — these
// are safe for concurrent use. Re-creating them per request would burn
// CPU on TLS handshakes for nothing.
var (
	cfg         Config
	s3Client    *s3.Client        // for direct S3 ops (Delete, etc.)
	s3Presigner *s3.PresignClient // for generating PUT URLs
	cfSigner    *sign.URLSigner   // for signing CloudFront GET URLs (RSA)
)

// Init bootstraps the package. Call once at app startup, after env is loaded.
//
// On EC2 with our IAM role attached, the AWS SDK auto-discovers credentials
// from the instance metadata service — no access keys needed. The same code
// works locally if you `aws configure` your laptop, OR if you set
// AWS_ACCESS_KEY_ID / AWS_SECRET_ACCESS_KEY env vars.
func Init(c Config) error {
	// ── Sanity-check required fields ──────────────────────
	// Crash early with a clear message rather than fail mysteriously
	// the first time a handler tries to presign.
	if c.Bucket == "" || c.Region == "" {
		return fmt.Errorf("photo: S3_BUCKET and S3_REGION must be set")
	}
	if c.CFDomain == "" || c.CFKeyPairID == "" || c.CFPrivateKeyPth == "" {
		return fmt.Errorf("photo: CF_DOMAIN, CF_KEY_PAIR_ID, CF_PRIVATE_KEY_PATH must be set")
	}

	// ── Load the CloudFront RSA private key from disk ─────
	// Done at startup (not lazily) for two reasons:
	//   1. Fail-fast: a bad/missing key file is a deploy bug, not a runtime
	//      one. We want the app to refuse to start.
	//   2. Performance: parsing a PEM file + decoding RSA is ~ms, not free.
	//      Doing it once vs. per-request matters under load.
	//
	// We don't use sign.LoadPEMPrivKeyFile because it only handles PKCS#1
	// ("BEGIN RSA PRIVATE KEY"), and modern openssl defaults to PKCS#8
	// ("BEGIN PRIVATE KEY"). Our loader tries both.
	privKey, err := loadRSAKey(c.CFPrivateKeyPth)
	if err != nil {
		return fmt.Errorf("photo: load CF private key from %s: %w", c.CFPrivateKeyPth, err)
	}

	// ── Load AWS SDK config ───────────────────────────────
	// LoadDefaultConfig walks a credential chain in this order:
	//   1. Env vars (AWS_ACCESS_KEY_ID etc.)
	//   2. ~/.aws/credentials (if `aws configure` was run)
	//   3. EC2 IMDS (instance metadata — our IAM role lives here)
	// First one that succeeds wins. Magic that makes "same code, different
	// places" actually work.
	awsCfg, err := config.LoadDefaultConfig(
		context.Background(),
		config.WithRegion(c.Region),
	)
	if err != nil {
		return fmt.Errorf("photo: load AWS config: %w", err)
	}

	// ── Build the clients ─────────────────────────────────
	// s3.NewFromConfig wraps awsCfg into a typed S3 client.
	s3Client = s3.NewFromConfig(awsCfg)

	// PresignClient is a SEPARATE thing built on top of the S3 client.
	// Its job: build pre-signed URLs without actually calling S3.
	s3Presigner = s3.NewPresignClient(s3Client)

	// CloudFront URL signer. Stateless once built — just holds the keyID
	// (a public identifier, e.g., K3LVUK5HGD8IW2) and the parsed RSA key.
	cfSigner = sign.NewURLSigner(c.CFKeyPairID, privKey)

	cfg = c
	return nil
}

// loadRSAKey reads a PEM-encoded RSA private key and parses it.
// Handles both PKCS#1 and PKCS#8 encodings — the difference is the
// PEM block type ("RSA PRIVATE KEY" vs "PRIVATE KEY") and the inner
// DER format. Modern openssl produces PKCS#8 by default.
func loadRSAKey(path string) (*rsa.PrivateKey, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}

	// pem.Decode strips the "-----BEGIN ... -----" wrapper and base64
	// decodes the contents, leaving raw DER bytes in block.Bytes.
	block, _ := pem.Decode(bytes)
	if block == nil {
		return nil, fmt.Errorf("not a PEM file (no -----BEGIN----- header)")
	}

	switch block.Type {
	case "RSA PRIVATE KEY":
		// PKCS#1 — RSA-specific format. Direct parse.
		return x509.ParsePKCS1PrivateKey(block.Bytes)
	case "PRIVATE KEY":
		// PKCS#8 — generic wrapper that holds an algorithm identifier
		// plus the inner key. We parse, then assert it's actually RSA
		// (could theoretically be EC, Ed25519, etc.).
		k, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parse pkcs8: %w", err)
		}
		rsaKey, ok := k.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("pkcs8 key is not RSA (CloudFront requires RSA)")
		}
		return rsaKey, nil
	default:
		return nil, fmt.Errorf("unsupported PEM block type %q", block.Type)
	}
}

// BuildKey constructs the S3 object key for an entity's profile photo.
//
// Format: photos/{entity}/{id}/profile{ext}
// e.g.,   photos/students/42/profile.jpg
//
// We keep `profile` as the filename (rather than something random) so
// each entity has at most ONE photo at a time. New upload → overwrites
// the old one. Versioning on the bucket means the old version isn't
// truly lost — it's just no longer the current one.
func BuildKey(entity string, id int, ext string) string {
	if ext == "" {
		ext = ".jpg"
	}
	return fmt.Sprintf("photos/%s/%d/profile%s", entity, id, ext)
}

// PresignUpload returns a URL the client can PUT to directly.
//
// The client uploads the image bytes straight to S3 using this URL —
// our server never sees the file body. That's the win: the app's
// bandwidth, memory, and CPU stay free for actual API work.
//
// Expiry: 15 minutes. Long enough for a slow upload on bad WiFi,
// short enough that a leaked URL becomes useless quickly.
func PresignUpload(ctx context.Context, key string) (string, error) {
	if s3Presigner == nil {
		return "", fmt.Errorf("photo: package not initialised — call Init at startup")
	}

	// PresignPutObject takes the same input as a regular PutObject call,
	// but instead of executing it, it builds a URL that — when PUT to —
	// performs that exact operation.
	req, err := s3Presigner.PresignPutObject(
		ctx,
		&s3.PutObjectInput{
			Bucket: &cfg.Bucket,
			Key:    &key,
		},
		// Functional options. PresignClient takes a slice of these.
		s3.WithPresignExpires(15*time.Minute),
	)
	if err != nil {
		return "", fmt.Errorf("photo: presign upload for %s: %w", key, err)
	}

	// req.URL contains the full signed URL with query parameters baked in
	// (Authorization-style: X-Amz-Algorithm, X-Amz-Date, X-Amz-Signature, etc.).
	return req.URL, nil
}

// PresignCloudFrontGet returns a CloudFront-signed URL the client can GET
// to fetch the object via the CDN.
//
// Different beast from PresignUpload:
//   - Signed with our RSA PRIVATE KEY, not AWS credentials
//   - Verified at the CloudFront edge using the matching PUBLIC KEY we
//     uploaded earlier (key group: studentsystemgo-key-group)
//   - URL goes through CloudFront's domain (d2pv7o3f0grsso.cloudfront.net),
//     not S3 directly. Edge caches the response.
//
// Expiry: 5 minutes. Short because READS are frequent — we generate a fresh
// URL on every page render. Tiny cost (no network call to sign) and tight
// expiry limits damage from a leaked URL.
func PresignCloudFrontGet(key string, ttl time.Duration) (string, error) {
	if cfSigner == nil {
		return "", fmt.Errorf("photo: package not initialised — call Init at startup")
	}
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}

	// Build the public-facing URL the user will request.
	// Note we use HTTPS — viewer protocol policy on the distribution
	// is "Redirect HTTP to HTTPS", but we save a redirect by going direct.
	rawURL := fmt.Sprintf("https://%s/%s", cfg.CFDomain, key)

	// Sign it. Behind the scenes:
	//   1. Build a "canned policy" JSON: {"Statement":[{"Resource":"<rawURL>",
	//      "Condition":{"DateLessThan":{"AWS:EpochTime":<expiry>}}}]}
	//   2. SHA-1 hash that JSON
	//   3. RSA-sign the hash with our private key
	//   4. Base64-encode (URL-safe) the signature
	//   5. Append ?Expires=<unix>&Signature=<base64>&Key-Pair-Id=<keyID>
	//
	// CloudFront edge does the inverse: parses the URL, looks up the public
	// key for Key-Pair-Id, verifies signature, checks expiry, then serves.
	signed, err := cfSigner.Sign(rawURL, time.Now().Add(ttl))
	if err != nil {
		return "", fmt.Errorf("photo: sign CF URL for %s: %w", key, err)
	}
	return signed, nil
}

// Delete removes the object at `key` from S3.
//
// Called when:
//   - User explicitly removes their photo (DELETE /students/{id}/photo)
//   - Entity is hard-deleted from DB (cleanup hook)
//
// Note on bucket versioning: we enabled it. So this DOES NOT permanently
// delete bytes — it adds a "delete marker." The previous version is still
// retrievable via S3 console for ~30 days (depending on lifecycle config).
// Effectively a soft-delete from the user's perspective; safe.
//
// Idempotent — deleting a non-existent key returns success (S3's choice).
// That's actually what we want: "make sure this thing isn't there" is
// safer than "fail if it isn't."
func Delete(ctx context.Context, key string) error {
	if s3Client == nil {
		return fmt.Errorf("photo: package not initialised — call Init at startup")
	}

	// DeleteObject is a REAL API call this time — no presigning. Our app's
	// IAM role does the deletion server-side, using the IAM creds resolved
	// at startup.
	_, err := s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: &cfg.Bucket,
		Key:    &key,
	})
	if err != nil {
		return fmt.Errorf("photo: delete %s: %w", key, err)
	}
	return nil
}
