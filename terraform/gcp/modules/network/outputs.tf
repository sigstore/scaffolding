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

output "network_name" {
  value = google_compute_network.network.name
}

output "network_self_link" {
  value = google_compute_network.network.self_link
}

output "subnetwork_self_link" {
  value = google_compute_subnetwork.subnetwork.self_link
}

output "secondary_ip_range" {
  value = google_compute_subnetwork.subnetwork.secondary_ip_range
}
