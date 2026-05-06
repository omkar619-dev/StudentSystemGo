# Input variables. Setting them in code (not a separate `terraform.tfvars`)
# because they don't change per environment — single-env project.
#
# If we later split into dev/prod, variables move into `dev.tfvars` /
# `prod.tfvars` and get loaded with `-var-file=dev.tfvars`.

variable "aws_region" {
  description = "Primary AWS region for non-CloudFront resources."
  type        = string
  default     = "ap-south-1"
}

variable "bucket_name" {
  description = "S3 bucket name for photo uploads. Must be globally unique."
  type        = string
  default     = "studentsystemgo-uploads-a7k3m2"
}

variable "ec2_role_name" {
  description = "IAM role attached to the EC2 instance running the app."
  type        = string
  default     = "studentsystemgo-ec2-role"
}

variable "s3_policy_name" {
  description = "IAM policy name for S3 photo access."
  type        = string
  default     = "studentsystemgo-s3-photos-access"
}

variable "cf_distribution_comment" {
  description = "Description shown in CloudFront console."
  type        = string
  default     = "studentsystemgo private photos via signed URLs"
}

variable "cf_oac_name" {
  description = "Origin Access Control name."
  type        = string
  default     = "studentsystemgo-oac"
}

variable "cf_public_key_name" {
  description = "CloudFront public key resource name."
  type        = string
  default     = "studentsystemgo-cf-signing-key"
}

variable "cf_key_group_name" {
  description = "CloudFront key group resource name."
  type        = string
  default     = "studentsystemgo-key-group"
}

variable "cf_public_key_pem_path" {
  description = "Local path to the PUBLIC key file (PEM). Private key stays out of git."
  type        = string
  default     = "../../secrets/cloudfront_public_key.pem"
}
