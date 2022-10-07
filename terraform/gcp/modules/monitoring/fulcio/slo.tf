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
  service_id            = "fulcio"
  display_name          = "Fulcio"
  resource_name         = format("//container.googleapis.com/projects/%s/locations/%s/clusters/%s/k8s/namespaces/%s", var.project_id, var.cluster_location, var.cluster_name, var.gke_namespace)
  notification_channels = local.notification_channels

  availability_slos = {
    server-availability = {
      display_prefix            = "Availability (Server)"
      base_total_service_filter = format("metric.type=\"prometheus.googleapis.com/grpc_server_handled_total/counter\" resource.type=\"prometheus_target\" resource.labels.namespace=\"%s\"", var.gke_namespace)
      # Only count server errors.
      # TODO: If clients can trigger DeadlineExceeded with short deadlines, reconsider this metric.
      bad_filter = "metric.labels.grpc_method=one_of(\"DeadlineExceeded\",\"Internal\")"
      slos = {
        all-methods = {
          display_suffix = "All Methods"
          label_filter   = ""
          goal           = 0.995
          label_filter   = "metric.labels.grpc_service=one_of(\"dev.sigstore.fulcio.v1beta.CA\",\"dev.sigstore.fulcio.v2.CA\")"
        },
        v1beta-create-signing-certificate = {
          display_suffix = "CreateSigningCertificate v1beta"
          label_filter   = "metric.labels.grpc_service=\"dev.sigstore.fulcio.v1beta.CA\" metric.labels.grpc_method=\"CreateSigningCertificate\""
          goal           = 0.995
        },
        v1beta-get-root-certificate = {
          display_suffix = "GetRootCertificate v1beta"
          label_filter   = "metric.labels.grpc_service=\"dev.sigstore.fulcio.v1beta.CA\" metric.labels.grpc_method=\"GetRootCertificate\""
          goal           = 0.995
        },
        v2-create-signing-certificate = {
          display_suffix = "CreateSigningCertificate v2"
          label_filter   = "metric.labels.grpc_service=\"dev.sigstore.fulcio.v2.CA\" metric.labels.grpc_method=\"CreateSigningCertificate\""
          goal           = 0.995
        },
        v2-get-configuration = {
          display_suffix = "GetConfiguration v2"
          label_filter   = "metric.labels.grpc_service=\"dev.sigstore.fulcio.v2.CA\" metric.labels.grpc_method=\"GetConfiguration\""
          goal           = 0.995
        },
        v2-get-trust-bundle = {
          display_suffix = "GetTrustBundle v2"
          label_filter   = "metric.labels.grpc_service=\"dev.sigstore.fulcio.v2.CA\" metric.labels.grpc_method=\"GetTrustBundle\""
          goal           = 0.995
        },
      }
    },
    prober-availability = {
      display_prefix            = "Availability (Prober)"
      base_total_service_filter = format("metric.type=\"prometheus.googleapis.com/api_endpoint_latency_count/summary\" resource.type=\"prometheus_target\" metric.labels.host=\"%s\"", var.prober_url)
      bad_filter                = "metric.labels.status_code!=monitoring.regex.full_match(\"20[0-1]\")"
      slos = {
        all-methods = {
          display_suffix = "All Methods"
          label_filter   = ""
          goal           = 0.995
        },
        v1beta-create-signing-certificate = {
          display_suffix = "CreateSigningCertificate v1beta (/api/v1/signingCert - POST)"
          label_filter   = "metric.labels.endpoint=\"/api/v1/signingCert\" metric.labels.method=\"POST\""
          goal           = 0.995
        },
        v1beta-get-root-certificate = {
          display_suffix = "GetRootCertificate v1beta (/api/v1/rootCert - GET)"
          label_filter   = "metric.labels.endpoint=\"/api/v1/rootCert\" metric.labels.method=\"GET\""
          goal           = 0.995
        },
        v2-create-signing-certificate = {
          display_suffix = "CreateSigningCertificate v2 (/api/v2/signingCert - POST)"
          label_filter   = "metric.labels.endpoint=\"/api/v2/signingCert\" metric.labels.method=\"POST\""
          goal           = 0.995
        },
        v2-get-configuration = {
          display_suffix = "GetConfiguration v2 (/api/v2/configuration - GET)"
          label_filter   = "metric.labels.endpoint=\"/api/v2/configuration\" metric.labels.method=\"GET\""
          goal           = 0.995
        },
        v2-get-trust-bundle = {
          display_suffix = "GetTrustBundle v2 (/api/v2/trustBundle - GET)"
          label_filter   = "metric.labels.endpoint=\"/api/v2/trustBundle\" metric.labels.method=\"GET\""
          goal           = 0.995
        },
      }
    }
  }
}

module "ctlog_slos" {
  source = "../slo"
  count  = var.create_slos ? 1 : 0

  project_id            = var.project_id
  service_id            = "ctlog"
  display_name          = "CT Log"
  resource_name         = format("//container.googleapis.com/projects/%s/locations/%s/clusters/%s/k8s/namespaces/%s", var.project_id, var.cluster_location, var.cluster_name, var.ctlog_gke_namespace)
  notification_channels = local.notification_channels

  availability_slos = {
    server-availability = {
      display_prefix            = "Availability (Server)"
      base_total_service_filter = format("metric.type=\"prometheus.googleapis.com/http_rsps/counter\" resource.type=\"prometheus_target\" resource.labels.namespace=\"%s\"", var.ctlog_gke_namespace)
      # Only count server errors.
      bad_filter = "metric.labels.rc=monitoring.regex.full_match(\"5[0-9][0-9]\")"
      slos = {
        all-methods = {
          display_suffix = "All Methods"
          label_filter   = ""
          goal           = 0.995
        },
        add-pre-chain = {
          display_suffix = "AddPreChain"
          label_filter   = "metric.labels.ep=\"AddPreChain\""
          goal           = 0.995
        }
      }
    }
  }
}
