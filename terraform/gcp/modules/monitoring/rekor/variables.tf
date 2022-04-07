variable "project_id" {
  type    = string
  default = ""
  validation {
    condition     = length(var.project_id) > 0
    error_message = "Must specify PROJECT_ID variable."
  }
}

// URLs for Sigstore services
variable "rekor_url" {
  description = "Rekor URL"
  default     = "rekor.sigstore.dev"
}

// Set-up for notification channel for alerting
variable "notification_channel_id" {
  type        = string
  description = "The notification channel ID which alerts should be sent to. You can find this by running `gcloud alpha monitoring channels list`."
}

locals {
  notification_channels = [format("projects/%v/notificationChannels/%v", var.project_id, var.notification_channel_id)]
}

