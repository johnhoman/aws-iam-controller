variable "role-name" {
  type = string
  default = "aws-iam-controller"
}

variable "principal" {
  type = map(string)
}

variable "account_id" {
  type = string
}
