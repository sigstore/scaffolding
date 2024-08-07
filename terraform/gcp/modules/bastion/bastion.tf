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

// Enable required services for this module
resource "google_project_service" "service" {
  for_each = toset([
    "cloudkms.googleapis.com", // For KMS keyring and crypto key. roles/cloudkms.admin
    "compute.googleapis.com",  // For compute firewall, instance. roles/compute.securityAdmin, roles/compute.instanceAdmin
    "iam.googleapis.com",      // For creating service accounts and access control. roles/iam.serviceAccountAdmin
    "osconfig.googleapis.com", // For using OS Config API (patching)
  ])
  service = each.key

  // Do not disable the service on destroy. On destroy, we are going to
  // destroy the project, but we need the APIs available to destroy the
  // underlying resources.
  disable_on_destroy = false
}

resource "random_id" "suffix" {
  byte_length = 4
}

// Dedicated service account for the Bastion instance
resource "google_service_account" "bastion" {
  account_id   = "bastion-${random_id.suffix.hex}"
  display_name = "Bastion"

  depends_on = [google_project_service.service]
}

resource "google_compute_firewall" "bastion-ingress" {
  // Only allow SSH access from Google identity aware proxies (e.g with the
  // magic of `gcloud compute ssh`)
  name      = "bastion-ingress-${random_id.suffix.hex}"
  network   = var.network
  direction = "INGRESS"

  // Identity-Aware proxy well-known ip address range
  // ref: https://cloud.google.com/iap/docs/using-tcp-forwarding#create-firewall-rule
  source_ranges = ["35.235.240.0/20"]

  allow {
    protocol = "tcp"
    ports    = ["22"]
  }

  target_tags = ["bastion"]

  depends_on = [google_project_service.service]
}

data "google_project" "project" {
}

resource "google_kms_crypto_key_iam_binding" "disk-key" {
  crypto_key_id = google_kms_crypto_key.disk-key.id
  role          = "roles/cloudkms.cryptoKeyEncrypterDecrypter"

  members = [
    "serviceAccount:service-${data.google_project.project.number}@compute-system.iam.gserviceaccount.com",
  ]
  depends_on = [google_kms_crypto_key.disk-key]
}

resource "google_kms_key_ring" "disk-keyring" {
  name       = "bastion-disk-keyring"
  location   = "global"
  depends_on = [google_project_service.service]
}

resource "google_kms_crypto_key" "disk-key" {
  name       = "bastion-disk-key"
  key_ring   = google_kms_key_ring.disk-keyring.id
  depends_on = [google_kms_key_ring.disk-keyring]

  lifecycle {
    prevent_destroy = true
  }
}

data "google_compute_zones" "available" {
  // All available AZ in our region
  region = var.region
}

resource "random_shuffle" "bastion_az" {
  // Randomly select an AZ from our region for the bastion
  input        = data.google_compute_zones.available.names
  result_count = 1
}

// The Bastion Host
resource "google_compute_instance" "bastion" {
  name         = "bastion-${random_id.suffix.hex}"
  machine_type = "g1-small"
  // coalesce function will choose whichever value isn't an empty string first
  zone = coalesce(var.zone, random_shuffle.bastion_az.result[0])
  tags = ["bastion"]

  boot_disk {
    kms_key_self_link = google_kms_crypto_key.disk-key.id
    initialize_params {
      image = "debian-cloud/debian-12"
    }
  }

  metadata = {
    block-project-ssh-keys = true
    enable-osconfig        = "TRUE"
  }

  shielded_instance_config {
    enable_secure_boot          = true
    enable_vtpm                 = true
    enable_integrity_monitoring = true
  }

  // Define a network interface in the correct subnet.
  network_interface {
    subnetwork = var.subnetwork

    // NB: No public ip address assigned. This bastion is not internet routable
    // and can only be accessed via Google identity aware proxies.
  }

  allow_stopping_for_update = true

  service_account {
    email = google_service_account.bastion.email
    // Bastion is strictly forbidden from talking to Google cloud APIs. We drop
    // all scopes.
    scopes = []
  }

  depends_on = [google_project_service.service, google_kms_crypto_key_iam_binding.disk-key]
}

resource "google_os_config_patch_deployment" "patch" {
  patch_deployment_id = "patch-deploy"

  instance_filter {
    instances = [google_compute_instance.bastion.id]
  }

  patch_config {
    apt {
      type = "DIST"
    }
  }

  recurring_schedule {
    time_zone {
      id = "Etc/UTC"
    }

    time_of_day {
      hours   = 0
      minutes = 0
      seconds = 0
      nanos   = 0
    }
  }

  depends_on = [google_project_service.service]
}

// Grant tunnel access to the GA team 
resource "google_project_iam_member" "ga_tunnel_accessor_verifier_member" {
  for_each = toset(var.tunnel_accessor_sa)

  project = var.project_id
  role    = "roles/iap.tunnelResourceAccessor"
  member  = each.key
}

// Grant access to impersonate the SA the bastion VM runs as
resource "google_service_account_iam_member" "bastion_access" {
  for_each = toset(var.tunnel_accessor_sa)

  service_account_id = google_service_account.bastion.name
  role               = "roles/iam.serviceAccountUser"
  member             = each.key
}
