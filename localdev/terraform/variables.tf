variable "ngrok_url" {
  default = "https://httpbin.org/post"
}

variable "tfc_organization" {
  description = "The Terraform Cloud organization where a test workspace will be created."
  type        = string
  sensitive   = false
}
