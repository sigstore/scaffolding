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

variable "cluster_name" {
  description = "The name to give the new Kubernetes cluster."
  type        = string
}

// Certificate authority
variable "ca_pool_name" {
  description = "Certificate authority pool name"
  type        = string
}
