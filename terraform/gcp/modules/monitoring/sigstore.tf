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
    "monitoring.googleapis.com", // For monitoring alerts.
  ])
  project = var.project_id
  service = each.key

  // Do not disable the service on destroy. On destroy, we are going to
  // destroy the project, but we need the APIs available to destroy the
  // underlying resources.
  disable_on_destroy = false
}

// Pull in submodules for rekor, fulcio, and dex

// Rekor
module "rekor" {
  source = "./rekor"

  project_id               = var.project_id
  notification_channel_ids = var.notification_channel_ids
  rekor_url                = var.rekor_url
  cluster_name             = var.cluster_name
  cluster_location         = var.cluster_location
  prober_url               = var.prober_rekor_url
  create_slos              = var.create_slos

  depends_on = [
    google_project_service.service
  ]
}

// Fulcio
module "fulcio" {
  source = "./fulcio"

  project_id               = var.project_id
  notification_channel_ids = var.notification_channel_ids
  ctlog_url                = var.ctlog_url
  fulcio_url               = var.fulcio_url
  cluster_name             = var.cluster_name
  cluster_location         = var.cluster_location
  prober_url               = var.prober_fulcio_url
  create_slos              = var.create_slos

  depends_on = [
    google_project_service.service
  ]
}

// Dex
module "dex" {
  source = "./dex"

  project_id               = var.project_id
  notification_channel_ids = var.notification_channel_ids
  cluster_name             = var.cluster_name
  cluster_location         = var.cluster_location
  create_slos              = var.create_slos

  dex_url = var.dex_url

  depends_on = [
    google_project_service.service
  ]
}

// Prober
module "prober" {
  source = "./prober"

  project_id               = var.project_id
  notification_channel_ids = var.notification_channel_ids
  rekor_url                = var.prober_rekor_url
  fulcio_url               = var.prober_fulcio_url

  depends_on = [
    google_project_service.service
  ]
}
// Infra
module "infra" {
  source = "./infra"

  project_id               = var.project_id
  notification_channel_ids = var.notification_channel_ids
  rekor_url                = local.qualified_rekor_url
  fulcio_url               = local.qualified_fulcio_url

  depends_on = [
    google_project_service.service
  ]
}
