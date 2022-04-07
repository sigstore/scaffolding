variable "project_id" {
  type    = string
  default = ""
  validation {
    condition     = length(var.project_id) > 0
    error_message = "Must specify project_id variable."
  }
}

variable "region" {
  description = "GCP region"
  type        = string
}

variable "cluster_name" {
  type    = string
  default = ""
}

variable "network" {
  type    = string
  default = "default"
}

// Storage
variable "attestation_bucket" {
  type        = string
  description = "Name of GCS bucket for attestation."
}


// KMS
variable "kms_keyring" {
  type        = string
  description = "Name of KMS keyring"
}

variable "kms_location" {
  type        = string
  description = "Location of KMS keyring"
}

variable "kms_key_name" {
  type        = string
  description = "Name of KMS key for Rekor"
}
