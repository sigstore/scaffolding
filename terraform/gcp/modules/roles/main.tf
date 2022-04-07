// Copyright 2022 The Sigstore Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

terraform {
  required_version = ">= 1.1.5"
  required_providers {
    google = {
      version = ">= 4.11.0"
    }
  }
}

// Enable required services for this module
resource "google_project_service" "service" {
  for_each = toset([
    "cloudresourcemanager.googleapis.com", // For IAM bindings.
  ])
  project = var.project_id
  service = each.key

  // Do not disable the service on destroy. On destroy, we are going to
  // destroy the project, but we need the APIs available to destroy the
  // underlying resources.
  disable_on_destroy = false
}

resource "google_project_iam_member" "membership" {
  project = var.project_id
  member  = "group:${var.member}@${var.domain}"

  for_each   = toset(var.roles)
  role       = each.value
  depends_on = [google_project_service.service]
}
