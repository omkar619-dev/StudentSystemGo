# S3 bucket for photo uploads.
#
# Hierarchy of resources (AWS provider 5.x split everything out):
#   1. aws_s3_bucket — the bucket itself
#   2. aws_s3_bucket_versioning — enable versioning (so deleted objects
#      can be restored — minor cost, big safety win)
#   3. aws_s3_bucket_server_side_encryption_configuration — SSE-S3 default
#   4. aws_s3_bucket_public_access_block — the four toggles that block
#      all anonymous access (defense in depth even if a bucket policy bug
#      tried to expose objects)
#
# The bucket policy that allows CloudFront to read is in cloudfront.tf —
# it depends on the distribution ARN, so it lives next to that resource.

resource "aws_s3_bucket" "uploads" {
  bucket = var.bucket_name

  # Don't auto-delete on destroy if non-empty — protects against
  # accidental `terraform destroy` wiping production photos.
  # If you actually need to destroy, set this to true, apply, destroy.
  force_destroy = false

  tags = {
    Purpose = "Profile photo uploads - private"
  }
}

# Versioning: keeps historical versions of objects when overwritten/deleted.
# Cost: storage for all versions. With our use case (one profile photo per
# entity, occasional updates), it's negligible.
resource "aws_s3_bucket_versioning" "uploads" {
  bucket = aws_s3_bucket.uploads.id

  versioning_configuration {
    status = "Enabled"
  }
}

# Server-side encryption with S3-managed keys (SSE-S3).
# Free, transparent, "encrypted at rest" is true.
# SSE-KMS would give us per-key audit/control but costs $1/key/month + per-call.
resource "aws_s3_bucket_server_side_encryption_configuration" "uploads" {
  bucket = aws_s3_bucket.uploads.id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
    bucket_key_enabled = true
  }
}

# Block all public access — four separate toggles that AWS surfaces
# as a single "Block public access" setting in the Console.
#
# These take precedence over bucket policies and ACLs. Even if someone
# wrote a bucket policy granting `Principal: "*"`, these blocks would
# silently override it. Belt-and-suspenders.
resource "aws_s3_bucket_public_access_block" "uploads" {
  bucket = aws_s3_bucket.uploads.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}
