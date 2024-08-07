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

output "ssh_cmd" {
  description = "Instructions to connect to bastion"
  value       = "gcloud compute ssh --zone ${google_compute_instance.bastion.zone} ${google_compute_instance.bastion.name} --tunnel-through-iap --project ${google_compute_instance.bastion.project}"
}

output "name" {
  value = google_compute_instance.bastion.name
}

output "zone" {
  value = google_compute_instance.bastion.zone
}

output "project" {
  value = google_compute_instance.bastion.project
}

output "ip_address" {
  description = "private IP address of bastion"
  value       = google_compute_instance.bastion.network_interface.0.network_ip
}

