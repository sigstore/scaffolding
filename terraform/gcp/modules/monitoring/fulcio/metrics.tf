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

# This file contains alerts generic to the Sigstore project
# Alerts specific to fulcio, rekor or dex should be in the appropriate `modules/monitoring/[service]` directory

resource "google_logging_metric" "fulcio_k8s_pod_restart_failing_container" {
  description = "Counts the number of k8s_pod resource logs that contain the \"restarting failed container\" message"
  filter      = "resource.labels.namespace_name=\"fulcio-system\"\nresource.type=k8s_pod AND severity>=WARNING\n\"Back-off restarting failed container\"\n"

  metric_descriptor {
    metric_kind = "DELTA"
    unit        = "1"
    value_type  = "INT64"
  }

  name    = "fulcio/k8s_pod/restarting-failed-container"
  project = var.project_id
}

resource "google_logging_metric" "k8s_pod_unschedulable" {
  description = "Counts the number of k8s_pod resource logs that contain the unschedulable message"
  filter      = "resource.labels.namespace_name=\"fulcio-system\"\nresource.type=k8s_pod AND severity>=WARNING\n\"unschedulable\"\n"

  metric_descriptor {
    metric_kind = "DELTA"
    unit        = "1"
    value_type  = "INT64"
  }

  name    = "fulcio/k8s_pod/unschedulable"
  project = var.project_id
}
