/**
 * Copyright 2025 The Sigstore Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */


resource "google_monitoring_uptime_check_config" "uptime_timestamp" {
  display_name = "Timestamp Authority Uptime"

  http_check {
    mask_headers   = "false"
    path           = "/api/v1/timestamp/certchain"
    port           = "443"
    request_method = "GET"
    use_ssl        = "true"
    validate_ssl   = "true"
  }

  monitored_resource {
    labels = {
      host       = var.timestamp_url
      project_id = var.project_id
    }

    type = "uptime_url"
  }

  period  = "60s"
  project = var.project_id
  timeout = "10s"
}
