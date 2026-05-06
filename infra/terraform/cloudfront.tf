# CloudFront distribution + signed-URL key pair + OAC for S3.
#
# All CloudFront resources MUST use the us-east-1 endpoint
# (CloudFront is a global service whose API only lives there).
# We use the aliased `aws.cloudfront` provider declared in main.tf.
#
# Five resources here:
#   1. aws_cloudfront_public_key       — the RSA public key used to verify signed URLs
#   2. aws_cloudfront_key_group        — wraps one or more public keys; rotation point
#   3. aws_cloudfront_origin_access_control — replaces legacy OAI; CF signs requests to S3
#   4. aws_cloudfront_distribution     — the actual CDN distribution
#   5. aws_s3_bucket_policy            — grants CF permission to read from the bucket
#
# Plus the matching `local_file` data source to read the public key PEM.

# ── 1. Read the public key PEM from disk ──────────────
# Pure-local — no AWS call. Just reads bytes from a file at apply time.
data "local_file" "cf_public_key" {
  filename = var.cf_public_key_pem_path
}

# ── 2. Upload the public key to CloudFront ────────────
resource "aws_cloudfront_public_key" "signing" {
  provider    = aws.cloudfront

  name        = var.cf_public_key_name
  comment     = "RSA public key for signing CloudFront URLs to private S3 bucket"
  encoded_key = data.local_file.cf_public_key.content

  # Ignore line-ending differences between local PEM (CRLF on Windows) and
  # AWS-stored content (LF). The key bytes themselves never change unless
  # we explicitly rotate. Without this, every plan would force-replace the
  # key, which would change its ID and break all signed URLs in prod.
  lifecycle {
    ignore_changes = [encoded_key]
  }
}

# ── 3. Group the public key (allows rotation) ─────────
# Distribution references key GROUPS, not keys directly. To rotate:
# add new key to group → wait for caches to expire → remove old key.
# No distribution change needed during rotation.
resource "aws_cloudfront_key_group" "signing" {
  provider = aws.cloudfront

  name    = var.cf_key_group_name
  comment = "Trusted signers for studentsystemgo private photo URLs"
  items   = [aws_cloudfront_public_key.signing.id]
}

# ── 4. Origin Access Control (OAC) ────────────────────
# Modern replacement for OAI (Origin Access Identity). CloudFront signs
# requests to S3 using SigV4 — your bucket policy validates the
# specific distribution is the caller. More secure than OAI which
# used a separate identity.
resource "aws_cloudfront_origin_access_control" "uploads" {
  provider = aws.cloudfront

  # Name + description below match what the CloudFront wizard auto-set
  # when we ticked "Allow private S3 bucket access" in the Console.
  # Renaming would force a destroy/replace which breaks CF->S3 access
  # briefly. Easier to match reality.
  name                              = "oac-studentsystemgo-uploads-a7k3m2.s3.ap-south-1.ama-moonvb1kz7y"
  description                       = "Created by CloudFront"
  origin_access_control_origin_type = "s3"
  signing_behavior                  = "always"
  signing_protocol                  = "sigv4"
}

# ── 5. The distribution ───────────────────────────────
resource "aws_cloudfront_distribution" "photos" {
  provider = aws.cloudfront

  enabled             = true
  is_ipv6_enabled     = true
  comment             = var.cf_distribution_comment
  default_root_object = ""

  # PriceClass_200 = NA + Europe + Asia + Middle East + Africa
  # (excludes Australia/South America to save ~30%)
  price_class = "PriceClass_200"

  origin {
    domain_name = aws_s3_bucket.uploads.bucket_regional_domain_name

    # The Console wizard auto-appended a "-moonpt2kcuy" random suffix
    # to origin_id. Matching it here so plan stays clean. Renaming would
    # destroy + recreate the origin (small risk window of broken state).
    origin_id                = "${aws_s3_bucket.uploads.bucket_regional_domain_name}-moonpt2kcuy"
    origin_access_control_id = aws_cloudfront_origin_access_control.uploads.id
  }

  default_cache_behavior {
    target_origin_id       = "${aws_s3_bucket.uploads.bucket_regional_domain_name}-moonpt2kcuy"
    viewer_protocol_policy = "redirect-to-https"
    allowed_methods        = ["GET", "HEAD"]
    cached_methods         = ["GET", "HEAD"]
    compress               = true

    # CachingOptimized is an AWS-managed policy. Its ID is well-known.
    # Hardcoding the ID is safe — it never changes.
    cache_policy_id = "658327ea-f89d-4fab-a63d-7e88639e58f6"

    # Signed URL enforcement — references the key group we created above.
    # Without this, distribution would serve to anyone who knows the URL.
    trusted_key_groups = [aws_cloudfront_key_group.signing.id]
  }

  # Geographic restrictions: none. Global access.
  restrictions {
    geo_restriction {
      restriction_type = "none"
    }
  }

  # TLS for viewers — using the default *.cloudfront.net cert.
  # Switch to a custom cert + ACM when we have a domain.
  viewer_certificate {
    cloudfront_default_certificate = true
  }

  # No HTTP version restrictions — CF auto-negotiates HTTP/2 + HTTP/3.
}

# ── 6. Bucket policy: allow CloudFront to GetObject ───
# This is what makes the OAC actually work. CF sends signed requests;
# bucket policy validates the SourceArn matches OUR distribution.
data "aws_iam_policy_document" "bucket" {
  statement {
    sid    = "AllowCloudFrontServicePrincipal"
    effect = "Allow"

    principals {
      type        = "Service"
      identifiers = ["cloudfront.amazonaws.com"]
    }

    actions   = ["s3:GetObject"]
    resources = ["${aws_s3_bucket.uploads.arn}/*"]

    condition {
      test     = "StringEquals"
      variable = "AWS:SourceArn"
      values   = [aws_cloudfront_distribution.photos.arn]
    }
  }
}

resource "aws_s3_bucket_policy" "uploads" {
  bucket = aws_s3_bucket.uploads.id
  policy = data.aws_iam_policy_document.bucket.json
}
