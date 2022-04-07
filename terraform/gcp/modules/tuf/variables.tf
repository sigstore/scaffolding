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
}

variable "tuf_bucket" {
  type        = string
  description = "Name of GCS bucket for TUF root."
}
