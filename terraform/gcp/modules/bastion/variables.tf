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

variable "network" {
  type        = string
  description = "VPC network to deploy bastion into"
  default     = "default"
}

variable "subnetwork" {
  type        = string
  description = "VPC subnetwork to deploy bastion into"
  default     = "default"
}

variable "tunnel_accessor_sa" {
  type        = string
  description = "Email of group to give access to the tunnel to"
}
