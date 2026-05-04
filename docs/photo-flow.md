# Profile Photo Upload — Architecture & Flow

## Overview

StudentSystemGo stores profile photos for students, teachers, and execs in
**Amazon S3**, served via **CloudFront** with **signed URLs**. The bucket is
fully private; no anonymous access exists.

Two design constraints drove the architecture:

1. **Server bandwidth must not scale with photo size.** A 100-user school
   uploading 5 MB photos shouldn't bottleneck on the API server's network.
2. **Reads must be fast and CDN-cached.** Profile photos render on every
   page load; serving them through the API server would burn both CPU and
   bandwidth.

The result is a pattern where the API server **never touches the photo
bytes** — it only signs URLs that let clients talk to S3 / CloudFront
directly.

---

## High-level architecture

```
┌────────────┐  signed-URL handshake   ┌─────────────┐
│            │ ───────────────────────▶│             │
│   Client   │                         │   API       │
│ (browser/  │ ◀───────────────────────│   server    │
│  app)      │      URLs / metadata    │   (Go)      │
└────┬───────┘                         └──────┬──────┘
     │                                        │
     │ direct PUT (upload)                    │ DeleteObject (admin only)
     │ direct GET (read)                      │ Get/Update photo_s3_key
     │                                        │
     ▼                                        ▼
┌────────────┐    OAC SigV4    ┌─────────────────────┐
│ CloudFront │ ───────────────▶│      S3 bucket      │
│   edge     │                 │  (private, SSE-S3,  │
│ (signed)   │ ◀───────────────│   versioning ON)    │
└────────────┘    image bytes  └─────────────────────┘
                                          ▲
                                          │
                                 ┌────────┴────────┐
                                 │   MariaDB        │
                                 │   students.      │
                                 │   photo_s3_key   │
                                 └──────────────────┘
```

**Components:**

| Component | Role |
|---|---|
| **API server** (Go) | Authn/authz; signs URLs; writes the S3 key into the DB after client confirms upload. Never handles photo bytes. |
| **S3 bucket** | Stores the actual image bytes. Private; only CloudFront (via OAC) and the EC2 IAM role can read. Versioning ON (cheap "undo"). |
| **CloudFront** | CDN in front of the bucket. Trusted Key Groups enforce signed URLs. OAC authenticates CF → S3. |
| **MariaDB** | Source of truth for "which student has which photo." Stores S3 key only — never bytes. |
| **EC2 IAM role** | Grants the Go app `s3:PutObject`, `s3:GetObject`, `s3:DeleteObject` on `photos/*`. Auto-rotated credentials via instance metadata. |
| **CloudFront RSA key pair** | Server holds private key (signs read URLs). CloudFront holds public key (verifies). |

---

## Data model

```sql
ALTER TABLE students  ADD COLUMN photo_s3_key VARCHAR(512) NULL;
ALTER TABLE teachers  ADD COLUMN photo_s3_key VARCHAR(512) NULL;
ALTER TABLE execs     ADD COLUMN photo_s3_key VARCHAR(512) NULL;
```

S3 keys follow the pattern:

```
photos/{entity}/{id}/profile{ext}
```

e.g. `photos/students/42/profile.jpg`. Each entity has at most ONE photo —
new uploads overwrite the same key. Bucket versioning preserves the prior
version for rollback.

---

## Flows

### 1. Upload (presign + PUT + confirm)

```
Client                  Server                    S3
  │                        │                       │
  │ POST /students/42/     │                       │
  │   photo/presign-upload │                       │
  │ {ext:".jpg"}           │                       │
  │ ──────────────────────▶│                       │
  │                        │ RBAC; validate id+ext │
  │                        │ key = "photos/        │
  │                        │   students/42/        │
  │                        │   profile.jpg"        │
  │                        │ presign PUT (15 min)  │
  │ ◀── {upload_url, key}──│                       │
  │                        │                       │
  │ PUT <upload_url>       │                       │
  │   <image bytes>        │                       │
  │ ──────────────────────────────────────────────▶│ S3 verifies sig,
  │                        │                       │ stores object.
  │ ◀── 200 OK ─────────────────────────────────── │
  │                        │                       │
  │ POST /students/42/     │                       │
  │   photo/confirm        │                       │
  │ {key:"photos/.../42/   │                       │
  │   profile.jpg"}        │                       │
  │ ──────────────────────▶│                       │
  │                        │ RBAC; HasPrefix guard │
  │                        │   stops ID-substitution│
  │                        │ UPDATE students SET   │
  │                        │   photo_s3_key = ?    │
  │ ◀── 204 No Content ────│                       │
```

**Why split presign + confirm?**

- **Presign is cheap** (local crypto, no DB).
- **Client may never actually upload** (network drop, user cancels).
  Splitting avoids polluting the DB with rows that point at non-existent
  S3 objects.
- **Confirm is the source-of-truth marker** — only after the client
  reports successful upload do we record the key.

This is the same pattern as Stripe Checkout, OAuth callbacks, and AWS
multipart uploads: *initiate → user does the slow thing → confirm*.

---

### 2. Read (signed CloudFront GET)

```
Client                  Server                    CloudFront           S3
  │                        │                          │                 │
  │ GET /students/42/photo │                          │                 │
  │ ──────────────────────▶│                          │                 │
  │                        │ RBAC                     │                 │
  │                        │ SELECT photo_s3_key      │                 │
  │                        │   FROM students          │                 │
  │                        │   (replica)              │                 │
  │                        │ RSA-sign URL (5 min TTL) │                 │
  │ ◀── {url, expires_in}──│                          │                 │
  │                        │                          │                 │
  │ GET <signed_cf_url>    │                          │                 │
  │ ──────────────────────────────────────────────────▶│                 │
  │                        │            CF verifies signature with      │
  │                        │            public key in trusted key group │
  │                        │                          │                 │
  │                        │            cache miss:   │                 │
  │                        │            ──────────────▶ OAC SigV4 ─────▶│
  │                        │            ◀────── image bytes ────────── │
  │                        │            cache + serve  │                │
  │                        │                          │                 │
  │ ◀── 200 + image bytes ─────────────────────────── │                 │
```

**Two layers of access control on reads:**

1. **Client → CloudFront** — signed URL must verify (RSA + expiry check).
   Done at the edge, before any origin call. Cache misses don't bypass.
2. **CloudFront → S3** — OAC (Origin Access Control) signs CF's request
   to S3 using SigV4. Bucket policy only allows this exact distribution.

Either layer failing returns 403; bytes never reach the client.

**Signature verified on every request, even cache hits.** Time-bounded
access only works if the bound is enforced per request — letting cache
hits skip the check would mean expired URLs serve forever.

---

### 3. Delete (DB-first, S3 best-effort)

```
Client                  Server               DB              S3
  │                        │                  │              │
  │ DELETE /students/42/   │                  │              │
  │   photo                │                  │              │
  │ ──────────────────────▶│                  │              │
  │                        │ RBAC             │              │
  │                        │ SELECT key ─────▶│ (replica)    │
  │                        │ ◀── key ─────────│              │
  │                        │                  │              │
  │                        │ STEP 1:          │              │
  │                        │ UPDATE photo_    │              │
  │                        │   s3_key = NULL ▶│ (primary)    │
  │                        │ ◀── ok ──────────│              │
  │                        │                  │              │
  │                        │ STEP 2:          │              │
  │                        │ DeleteObject ────┼─────────────▶│
  │                        │   (best-effort)  │              │
  │                        │ failure → log,   │              │
  │                        │   still 204      │              │
  │ ◀── 204 No Content ────│                  │              │
```

**Why DB-first?** When two systems must stay in sync, decide which side
is allowed to be wrong on partial failure:

- **DB-first → S3 fails**: row says "no photo," S3 has orphan bytes.
  User-visible state: gone. Storage waste: small, reapable via lifecycle.
- **S3-first → DB fails**: row points to deleted key. Client GET → CF
  fetches from S3 → 404 → broken image icon. Worst case is user-visible.

**Rule: prefer failures that look like "nothing happened" over failures
that look like "something is broken."**

DELETE is also **idempotent** — calling it on an entity with no photo
returns 204, not 404. Lets clients retry safely without branching logic.

---

## Cryptography summary

| Operation | Algorithm | Signed by | Verified by | URL parameter |
|---|---|---|---|---|
| S3 PUT (upload) | HMAC-SHA256 (SigV4) | EC2 IAM role's temp creds | S3 | `X-Amz-Signature` |
| CloudFront GET (read) | RSA-SHA1 | App's RSA private key | CloudFront edge using public key from trusted key group | `Signature` + `Key-Pair-Id` |
| OAC (CF → S3 hop) | HMAC-SHA256 (SigV4) | CloudFront service | S3 | (internal) |

The two viewer-facing schemes use different crypto for a reason:
- **S3 trusts AWS itself** → symmetric SigV4 with the requester's IAM creds is enough.
- **CloudFront has no authority over your bucket** → needs to prove it via
  an asymmetric signature anyone can verify with the public key.

---

## Security model

### What each secret protects

| Stolen item | Worst-case impact |
|---|---|
| **CloudFront private key** | Read forgery — attacker can sign URLs to view any photo. Cannot delete, upload, or list. Recoverable: rotate the key group. |
| **EC2 IAM role credentials** (would require host compromise) | Full S3 access on `photos/*` — read, write, delete. Recoverable: revoke role, restore from S3 versioning. |
| **CloudFront Public Key ID** (`K3LVUK5HGD8IW2`) | Nothing. Public identifier. |
| **S3 bucket name** | Nothing — bucket is private. |

### Defenses against parameter tampering (IDOR)

The `confirm` endpoint checks that the client-supplied S3 key has the
expected prefix for the URL's entity ID:

```go
expectedPrefix := fmt.Sprintf("photos/students/%d/profile", id)
if !strings.HasPrefix(body.Key, expectedPrefix) {
    return 400
}
```

Without this, a logged-in user could call
`POST /students/42/photo/confirm` with body
`{"key":"photos/execs/1/profile.jpg"}` and assign exec 1's photo to
student 42's profile.

**General lesson: anytime a client passes back something the server gave
them earlier, the server must re-verify it matches the current request's
authorization context.**

### File-content trust

Extension whitelist (`.jpg`, `.jpeg`, `.png`, `.webp`) is enforced on
presign. **Bytes are NOT validated post-upload** — a determined attacker
could PUT a polyglot file (valid as both image and HTML/JS).

For production, add server-side content sniffing after upload (read
first KB from S3, check magic numbers) plus
`Content-Disposition: attachment` headers to force download instead of
inline render.

---

## Why this design (tradeoff log)

| Decision | Alternative | Why we chose this |
|---|---|---|
| Presigned PUT (client → S3 direct) | App proxies upload | Server bandwidth doesn't scale with photo size. |
| CloudFront with signed URLs | Direct S3 presigned GET | Edge caching: subsequent reads served from PoPs near the user, not Mumbai. |
| Private bucket + OAC | Public bucket | Defense in depth; bucket can never accidentally serve to anonymous viewers. |
| Two-step upload (presign + confirm) | Single-step | Don't write DB until upload actually succeeded. |
| DB-first delete order | S3-first | Optimize for "user-visible state correct" > "storage perfectly clean." |
| Key format `photos/{entity}/{id}/profile.{ext}` | Random key per upload | Single photo per entity; new uploads overwrite. Versioning preserves history. |
| 15-min upload TTL | Longer | Limit window of leaked-URL abuse. Plenty for slow uploads. |
| 5-min read TTL | Longer | Re-signing is free (local crypto); short TTL minimizes leak risk. |
| RSA-SHA1 for CF | RSA-SHA256 | CloudFront's only canned-policy option. SHA1 is fine for short-TTL signing. |
| EC2 IAM role | Long-lived access keys in env | Auto-rotated temp creds; no key file in repo or `.env`. |

---

## Endpoints

All endpoints require JWT auth (cookie or `Authorization: Bearer`).

| Method | Path | Roles | Description |
|---|---|---|---|
| POST | `/{entity}/{id}/photo/presign-upload` | admin, manager, exec | Returns presigned S3 PUT URL + S3 key |
| POST | `/{entity}/{id}/photo/confirm` | admin, manager, exec | Records uploaded key in DB |
| GET | `/{entity}/{id}/photo` | admin, manager, exec, teacher | Returns short-lived CloudFront signed URL |
| DELETE | `/{entity}/{id}/photo` | admin, manager, exec | Clears DB key, deletes S3 object (best-effort) |

`{entity}` ∈ `students`, `teachers`, `execs`.

---

## Configuration (env vars)

| Variable | Example | Notes |
|---|---|---|
| `S3_BUCKET` | `studentsystemgo-uploads-a7k3m2` | Globally unique bucket |
| `S3_REGION` | `ap-south-1` | Match EC2 region (no cross-region traffic) |
| `CF_DOMAIN` | `d2pv7o3f0grsso.cloudfront.net` | No scheme |
| `CF_KEY_PAIR_ID` | `K3LVUK5HGD8IW2` | Public identifier; safe in env |
| `CF_PRIVATE_KEY_PATH` | `/run/secrets/cloudfront_private_key.pem` | RSA private key on disk; mount as Docker secret in prod |

---

## Future improvements

- **Content-type sniffing** post-upload (magic-byte check) to prevent polyglot abuse.
- **Image resizing** via Lambda@Edge or a CloudFront function — generate thumbnails on-the-fly.
- **S3 lifecycle policy** to reap orphaned objects (no DB row pointing to them) after N days.
- **Periodic cleanup job** that diffs `SELECT photo_s3_key FROM students UNION ...` against S3's listing and deletes orphans.
- **Key rotation** — add new RSA key pair to the trusted key group, switch app to sign with new private key, retire old key after old signed URLs expire.
- **CloudFront access logs** to S3 for audit (currently disabled to save cost).
