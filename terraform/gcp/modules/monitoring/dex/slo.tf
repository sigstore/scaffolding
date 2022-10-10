/**
 * Copyright 2022 The Sigstore Authors
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

module "slos" {
  source = "../slo"
  count  = var.create_slos ? 1 : 0

  project_id            = var.project_id
  service_id            = "dex"
  display_name          = "Dex"
  resource_name         = format("//container.googleapis.com/projects/%s/locations/%s/clusters/%s/k8s/namespaces/%s", var.project_id, var.cluster_location, var.cluster_name, var.gke_namespace)
  notification_channels = local.notification_channels

  availability_slos = {
    server-availability = {
      display_prefix            = "Availability (Server)"
      base_total_service_filter = format("metric.type=\"prometheus.googleapis.com/http_requests_total/counter\" resource.type=\"prometheus_target\" resource.labels.namespace=\"%s\"", var.gke_namespace)
      # Only count server errors.
      bad_filter = "metric.labels.code=monitoring.regex.full_match(\"5[0-9][0-9]\")"
      slos = {
        all-methods = {
          display_suffix = "All Methods"
          label_filter   = "metric.labels.handler!=\"healthz\""
          goal           = 0.995
        },
      }
    }
  }
}
