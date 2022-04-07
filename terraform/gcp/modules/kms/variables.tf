variable "project_id" {
  type    = string
  default = ""
  validation {
    condition     = length(var.project_id) > 0
    error_message = "Must specify project_id variable."
  }
}

variable "region" {
  type        = string
  description = "GCP region"
  default     = "us-west1"
}

variable "cluster_name" {
  type    = string
  default = "sigstore-staging"
}

variable "name" {
  description = "KMS KeyRing name"
  type        = string
  default     = "rekor-keyring"
}

variable "location" {
  type    = string
  default = "global"
}

variable "key_name" {
  description = "KMS Key name"
  type        = string
  default     = "rekor-key"
}
