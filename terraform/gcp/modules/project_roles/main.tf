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

locals {
  // Flatten the member to roles map to a list so we can use
  // for_each expansion for each role binding.
  iam_member_role_list = flatten([
    for member, role_list in var.iam_members_to_roles : [
      for role in role_list : {
        member = member
        role   = role
      }
    ]
  ])
}

resource "google_project_iam_member" "membership" {
  project = var.project_id
  // Use the "<member> <role>" as the unique key for each binding. Neither members
  // nor roles can contain whitespace so this is guaranteed to be unique.
  for_each = {
    for member_role_binding in local.iam_member_role_list :
    "${member_role_binding.member} ${member_role_binding.role}" => member_role_binding
  }
  member = each.value.member

  role       = each.value.role
  depends_on = [google_project_service.service]
}
