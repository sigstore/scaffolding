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
  service_id            = "rekor"
  display_name          = "Rekor"
  resource_name         = format("//container.googleapis.com/projects/%s/locations/%s/clusters/%s/k8s/namespaces/%s", var.project_id, var.cluster_location, var.cluster_name, var.gke_namespace)
  notification_channels = local.notification_channels

  availability_slos = {
    server-availability = {
      display_prefix            = "Availability (Server)"
      base_total_service_filter = "metric.type=\"prometheus.googleapis.com/rekor_qps_by_api/counter\" resource.type=\"prometheus_target\""
      # Only count 500s as server errors since clients can trigger 400s.
      bad_filter = "metric.labels.code=monitoring.regex.full_match(\"5[0-9][0-9]\")"
      slos = {
        api-v1-all-methods = {
          display_suffix = "All Methods"
          label_filter   = "metric.labels.path!=\"/api/v1/index/retrieve\" metric.labels.method=\"GET\""
          goal           = 0.995
        },
        api-v1-log-get = {
          display_suffix = "/api/v1/log - GET"
          label_filter   = "metric.labels.path=\"/api/v1/log\" metric.labels.method=\"GET\""
          goal           = 0.995
        },
        api-v1-log-entries-get = {
          display_suffix = "/api/v1/log/entries - GET"
          label_filter   = "metric.labels.path=\"/api/v1/log/entries\" metric.labels.method=\"GET\""
          goal           = 0.995
        },
        api-v1-log-entries-post = {
          display_suffix = "/api/v1/log/entries - POST"
          label_filter   = "metric.labels.path=\"/api/v1/log/entries\" metric.labels.method=\"POST\""
          goal           = 0.995
        },
        api-v1-log-entries-uuid-get = {
          display_suffix = "/api/v1/log/entries/{entryUUID} - GET"
          label_filter   = "metric.labels.path=\"/api/v1/log/entries/{entryUUID}\" metric.labels.method=\"GET\""
          goal           = 0.995
        },
        api-v1-log-proof-get = {
          display_suffix = "/api/v1/log/proof - GET"
          label_filter   = "metric.labels.path=\"/api/v1/log/proof\" metric.labels.method=\"GET\""
          goal           = 0.995
        },
        api-v1-log-entries-retrieve-post = {
          display_suffix = "/api/v1/log/entries/retrieve - POST"
          label_filter   = "metric.labels.path=\"/api/v1/log/entries/retrieve\" metric.labels.method=\"POST\""
          goal           = 0.995
        },
        api-v1-index-retrieve-post = {
          display_suffix = "/api/v1/index/retrieve - POST"
          label_filter   = "metric.labels.path=\"/api/v1/index/retrieve\" metric.labels.method=\"POST\""
          goal           = 0.8
          page_configuration = {
            fast_burn   = false
            medium_burn = false
            slow_burn   = false
          }
        },
      }
    },
    prober-availability = {
      display_prefix            = "Availability (Prober)"
      base_total_service_filter = format("metric.type=\"prometheus.googleapis.com/api_endpoint_latency_count/summary\" resource.type=\"prometheus_target\" metric.labels.host=\"%s\"", var.prober_url)
      bad_filter                = "metric.labels.status_code!=monitoring.regex.full_match(\"20[0-1]\")"
      slos = {
        api-v1-all-methods = {
          display_suffix = "All Methods"
          # exclude unsupported APIs in total availability SLO - data still exists for "version" API
          label_filter = "metric.labels.endpoint!=\"/api/v1/version\" metric.labels.endpoint!=\"/api/v1/index/retrieve\" metric.labels.method=\"GET\""
          goal         = 0.995
        },
        api-v1-index-retrieve-post = {
          display_suffix = "/api/v1/index/retrieve - POST"
          label_filter   = "metric.labels.endpoint=\"/api/v1/index/retrieve\" metric.labels.method=\"POST\""
          goal           = 0.8
          page_configuration = {
            fast_burn   = false
            medium_burn = false
            slow_burn   = false
          }
        },
        api-v1-log-get = {
          display_suffix = "/api/v1/log - GET"
          label_filter   = "metric.labels.endpoint=\"/api/v1/log\" metric.labels.method=\"GET\""
          goal           = 0.995
        },
        api-v1-log-entries-get = {
          display_suffix = "/api/v1/log/entries - GET"
          label_filter   = "metric.labels.endpoint=\"/api/v1/log/entries\" metric.labels.method=\"GET\""
          goal           = 0.995
        },
        api-v1-log-entries-post = {
          display_suffix = "/api/v1/log/entries - POST"
          label_filter   = "metric.labels.endpoint=\"/api/v1/log/entries\" metric.labels.method=\"POST\""
          goal           = 0.995
        },
        api-v1-log-entries-uuid-get = {
          display_suffix = "/api/v1/log/entries/{entryUUID} - GET"
          label_filter   = "metric.labels.endpoint=\"/api/v1/log/entries/{entryUUID}\" metric.labels.method=\"GET\""
          goal           = 0.995
        },
        api-v1-log-entries-retrieve-post = {
          display_suffix = "/api/v1/log/entries/retrieve - POST"
          label_filter   = "metric.labels.endpoint=\"/api/v1/log/entries/retrieve\" metric.labels.method=\"POST\""
          goal           = 0.995
        },
        api-v1-log-proof-get = {
          display_suffix = "/api/v1/log/proof - GET"
          label_filter   = "metric.labels.endpoint=\"/api/v1/log/proof\" metric.labels.method=\"GET\""
          goal           = 0.995
        },
        api-v1-log-publickey-get = {
          display_suffix = "/api/v1/log/publicKey - GET"
          label_filter   = "metric.labels.endpoint=\"/api/v1/log/publicKey\" metric.labels.method=\"GET\""
          goal           = 0.995
        },
      }
    }
  }
}
