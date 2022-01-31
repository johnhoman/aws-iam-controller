variable "account_id" {
  type = string
}

variable "oidc-issuer" {
  type = string
}

variable "namespace" {
  type = string
  default = "aws-iam-controller-system"
}

variable "role-name" {
  type = string
  default = "aws-iam-controller"
}

variable "service_account" {
  type = string
  default = "aws-iam-controller-controller-manager"
}
