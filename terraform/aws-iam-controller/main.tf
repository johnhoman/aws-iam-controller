terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 3.0"
    }
  }
}

provider "aws" {}

resource "aws_iam_role_policy" "aws_iam_controller_policy" {
  name = "aws-iam-controller-policy"
  role = aws_iam_role.aws_iam_controller.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Action = [
        "iam:GetRole",
        "iam:UpdateAssumeRolePolicy",
        "iam:DetachRolePolicy",
        "iam:ListAttachedRolePolicies",
        "iam:CreateRole",
        "iam:DeleteRole",
        "iam:AttachRolePolicy",
        "iam:UpdateRole",
        "iam:ListRolePolicies",
        "iam:GetRolePolicy",
      ]
      Effect = "Allow"
      Resource = ["arn:aws:iam::${var.account_id}:role/*"]
    },{
      Action = ["iam:ListRoles"]
      Effect = "Allow"
      Resource = ["*"]

    }]
  })
}

resource "aws_iam_role" "aws_iam_controller" {
  name = var.role-name
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Action = "sts:AssumeRole"
      Effect = "Allow"
      Principal = var.principal
    }]
  })
}

