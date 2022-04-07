resource "google_monitoring_uptime_check_config" "uptime_fulcio" {
  display_name = "Fulcio Uptime"

  http_check {
    mask_headers   = "false"
    path           = "/api/v1/rootCert"
    port           = "443"
    request_method = "GET"
    use_ssl        = "true"
    validate_ssl   = "true"
  }

  monitored_resource {
    labels = {
      host       = var.fulcio_url
      project_id = var.project_id
    }

    type = "uptime_url"
  }

  period  = "60s"
  project = var.project_id
  timeout = "10s"
}

resource "google_monitoring_uptime_check_config" "uptime_ct_log" {
  display_name = "CT Log Uptime"

  http_check {
    mask_headers   = "false"
    path           = "/"
    port           = "80"
    request_method = "GET"
    use_ssl        = "false"
    validate_ssl   = "false"
  }

  monitored_resource {
    labels = {
      cluster_name   = var.cluster_name
      location       = var.cluster_location
      namespace_name = "fulcio-system"
      project_id     = var.project_id
      service_name   = "ct-log"
    }

    type = "k8s_service"
  }

  period  = "60s"
  project = var.project_id
  timeout = "10s"
}
