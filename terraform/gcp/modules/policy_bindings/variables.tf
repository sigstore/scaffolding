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
  default = ""
}

variable "github_repo" {
  type    = string
  default = ""
}

variable "subnetwork" {
  type    = string
  default = "default"
}
