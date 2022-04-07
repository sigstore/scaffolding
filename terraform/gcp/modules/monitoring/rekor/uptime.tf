resource "google_monitoring_uptime_check_config" "uptime_rekor" {
  display_name = "Rekor uptime"

  http_check {
    mask_headers   = "false"
    path           = "/"
    port           = "443"
    request_method = "GET"
    use_ssl        = "true"
    validate_ssl   = "true"
  }

  monitored_resource {
    labels = {
      host       = var.rekor_url
      project_id = var.project_id
    }

    type = "uptime_url"
  }

  period  = "60s"
  project = var.project_id
  timeout = "10s"
}
