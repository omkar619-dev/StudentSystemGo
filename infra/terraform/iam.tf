# IAM resources for the EC2 instance to access S3.
#
# Hierarchy:
#   1. Trust policy doc — "EC2 service can assume this role"
#   2. Permission policy doc — "this role can put/get/delete on photos/* in our bucket"
#   3. Role — combines (1) as trust relationship + has a name
#   4. Policy — wraps (2) as a standalone IAM policy
#   5. Attachment — links role + policy
#   6. Instance profile — the wrapper EC2 actually attaches to instances
#
# Why so many resources for one logical thing?
# IAM separates concepts strictly: roles, policies, attachments, profiles.
# Lets you reuse policies across roles, swap policies without touching roles,
# audit each piece independently. Verbose but flexible.

# ── 1. Trust policy: who can assume this role ─────────
# `data` blocks read or compute things — they don't create AWS resources.
# This one builds an IAM policy JSON in memory; we feed it to the role below.
data "aws_iam_policy_document" "ec2_assume" {
  statement {
    effect  = "Allow"
    actions = ["sts:AssumeRole"]

    principals {
      type        = "Service"
      identifiers = ["ec2.amazonaws.com"]
    }
  }
}

# ── 2. Permission policy: what the role can DO ────────
# Only s3:Put/Get/DeleteObject, only on photos/* in our bucket.
# Principle of least privilege.
data "aws_iam_policy_document" "s3_photos" {
  statement {
    sid    = "PhotosObjectAccess"
    effect = "Allow"
    actions = [
      "s3:PutObject",
      "s3:GetObject",
      "s3:DeleteObject",
    ]
    # ${aws_s3_bucket.uploads.arn} = "arn:aws:s3:::studentsystemgo-uploads-a7k3m2"
    # Suffix /photos/* scopes us to objects under photos/, not the whole bucket.
    resources = ["${aws_s3_bucket.uploads.arn}/photos/*"]
  }
}

# ── 3. The role itself ────────────────────────────────
resource "aws_iam_role" "ec2_app" {
  name               = var.ec2_role_name
  description        = "Attached to the t3.micro running StudentSystemGo. Grants S3 photo access."
  assume_role_policy = data.aws_iam_policy_document.ec2_assume.json
}

# ── 4. The policy ─────────────────────────────────────
resource "aws_iam_policy" "s3_photos" {
  name        = var.s3_policy_name
  description = "Allows put get delete on photos in the uploads bucket. Used by the EC2 app role"
  policy      = data.aws_iam_policy_document.s3_photos.json
}

# ── 5. Attach policy to role ──────────────────────────
resource "aws_iam_role_policy_attachment" "ec2_s3_photos" {
  role       = aws_iam_role.ec2_app.name
  policy_arn = aws_iam_policy.s3_photos.arn
}

# ── 6. Instance profile ───────────────────────────────
# AWS oddity: EC2 doesn't directly attach a "role". It attaches an
# "instance profile" which is a thin wrapper around exactly one role.
# Console hides this — you "attach a role to an instance" in the UI,
# but under the hood AWS creates the matching instance profile.
#
# When we click-created the role earlier, AWS auto-made an instance
# profile with the same name. So we name ours identically to match.
resource "aws_iam_instance_profile" "ec2_app" {
  name = var.ec2_role_name # same name as role — the convention
  role = aws_iam_role.ec2_app.name
}
