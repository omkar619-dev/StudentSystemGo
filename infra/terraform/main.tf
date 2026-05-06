# Terraform & provider configuration for StudentSystemGo's AWS infrastructure.
#
# Manages: S3 bucket for photo uploads, IAM role for EC2 to access S3,
# CloudFront distribution + signed-URL key group for private photo delivery.
#
# State backend: local for now. Production setups would use S3 + DynamoDB
# for state locking (so multiple devs don't apply concurrently). Keeping
# local until there's a real second person on the project.

terraform {
  required_version = ">= 1.6"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0" # 5.x major. Pinned to avoid breaking changes.
    }
  }

  # Uncomment when ready for remote state:
  # backend "s3" {
  #   bucket         = "studentsystemgo-tf-state-<unique>"
  #   key            = "infra/terraform.tfstate"
  #   region         = "ap-south-1"
  #   dynamodb_table = "studentsystemgo-tf-locks"
  #   encrypt        = true
  # }
}

# AWS provider — uses credentials from your laptop's `aws configure`
# (or env vars / instance profile). No keys hardcoded here.
provider "aws" {
  region = var.aws_region

  # Tag every resource we create with these. Lets us find/audit
  # everything Terraform owns vs. things created via Console.
  default_tags {
    tags = {
      Project   = "StudentSystemGo"
      ManagedBy = "Terraform"
      Owner     = "omkar"
    }
  }
}

# CloudFront-specific provider alias.
# CloudFront is a global service — its API only lives in us-east-1.
# Even though our buckets/EC2 are in ap-south-1, CF resources MUST be
# created via the us-east-1 endpoint. This alias provides that.
provider "aws" {
  alias  = "cloudfront"
  region = "us-east-1"

  default_tags {
    tags = {
      Project   = "StudentSystemGo"
      ManagedBy = "Terraform"
      Owner     = "omkar"
    }
  }
}
