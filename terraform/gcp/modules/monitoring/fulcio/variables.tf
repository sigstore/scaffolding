variable "project_id" {
  type    = string
  default = ""
  validation {
    condition     = length(var.project_id) > 0
    error_message = "Must specify PROJECT_ID variable."
  }
}

variable "cluster_location" {
  type        = string
  description = "Zone or Region to create cluster in."
  default     = "us-central1"
}

// Optional values that can be overridden or appended to if desired.
variable "cluster_name" {
  description = "The name of the Kubernetes cluster."
  type        = string
  default     = ""
}

// URLs for Sigstore services
variable "fulcio_url" {
  description = "Fulcio URL"
  default     = "fulcio.sigstore.dev"
}


// Set-up for notification channel for alerting
variable "notification_channel_id" {
  type        = string
  description = "The notification channel ID which alerts should be sent to. You can find this by running `gcloud alpha monitoring channels list`."
}

locals {
  notification_channels = [format("projects/%v/notificationChannels/%v", var.project_id, var.notification_channel_id)]
}

// Certificate Authority name for alerting
variable "ca_pool_name" {
  description = "Certificate authority pool name"
  type        = string
  default     = "sigstore"
}
