variable "project_id" {
  type = string
  validation {
    condition     = length(var.project_id) > 0
    error_message = "Must specify project_id variable."
  }
}

variable "region" {
  description = "The region in which to create the VPC network"
  type        = string
}

variable "attestation_bucket" {
  type        = string
  description = "Name of GCS bucket for attestation."
}

variable "tuf_bucket" {
  type        = string
  description = "Name of GCS bucket for TUF root."
}

variable "ca_pool_name" {
  description = "Certificate authority pool name"
  type        = string
  default     = "sigstore"
}

variable "monitoring" {
  description = "Monitoring and alerting"
  type = object({
    enabled                 = bool
    fulcio_url              = string
    rekor_url               = string
    dex_url                 = string
    notification_channel_id = string
  })
  default = {
    enabled                 = false
    fulcio_url              = "fulcio.example.com"
    rekor_url               = "rekor.example.com"
    dex_url                 = "oauth2.example.com"
    notification_channel_id = ""
  }
}

// Optional values that can be overridden or appended to if desired.
variable "cluster_name" {
  description = "The name to give the new Kubernetes cluster."
  type        = string
  default     = "sigstore-staging"
}

variable "tunnel_accessor_sa" {
  type        = string
  description = "Email of group to give access to the tunnel to"
}

variable "github_repo" {
  description = "Github repo for running Github Actions from."
  type        = string
}
