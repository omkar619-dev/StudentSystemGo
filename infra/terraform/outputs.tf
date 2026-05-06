# Outputs — values you'll need elsewhere (.env files, CI scripts, etc.).
# `terraform output <name>` prints any of these from the command line.

output "bucket_name" {
  description = "S3 bucket holding photo uploads."
  value       = aws_s3_bucket.uploads.bucket
}

output "ec2_role_name" {
  description = "IAM role attached to the EC2 instance for S3 access."
  value       = aws_iam_role.ec2_app.name
}

output "ec2_instance_profile_name" {
  description = "Instance profile name (what EC2 actually attaches)."
  value       = aws_iam_instance_profile.ec2_app.name
}

output "cloudfront_domain" {
  description = "CloudFront distribution domain. Goes into CF_DOMAIN env var."
  value       = aws_cloudfront_distribution.photos.domain_name
}

output "cloudfront_distribution_id" {
  description = "Distribution ID. Used for cache invalidations."
  value       = aws_cloudfront_distribution.photos.id
}

output "cloudfront_key_pair_id" {
  description = "Public key ID. Goes into CF_KEY_PAIR_ID env var; used in signed URLs."
  value       = aws_cloudfront_public_key.signing.id
}

output "cloudfront_key_group_id" {
  description = "Key group ID. Referenced by distribution; rotation point."
  value       = aws_cloudfront_key_group.signing.id
}
