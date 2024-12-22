/**
 * Copyright 2024 The Sigstore Authors
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

resource "google_sql_database" "searchindexes" {
  name      = "searchindexes"
  project   = var.project_id
  instance  = var.index_database_instance_name
  collation = "utf8mb3_general_ci"
}

// be sure to manually GRANT SELECT, INSERT, CREATE privileges for this user
resource "google_sql_user" "iam_user" {
  name     = google_service_account.rekor-sa.email
  instance = var.index_database_instance_name
  type     = "CLOUD_IAM_SERVICE_ACCOUNT"
}

resource "google_project_iam_member" "db_admin_member_rekor" {
  project    = var.project_id
  role       = "roles/cloudsql.client"
  member     = "serviceAccount:${google_service_account.rekor-sa.email}"
  depends_on = [google_service_account.rekor-sa]
}

resource "google_project_iam_member" "db_iam_auth" {
  project    = var.project_id
  role       = "roles/cloudsql.instanceUser"
  member     = "serviceAccount:${google_service_account.rekor-sa.email}"
  depends_on = [google_service_account.rekor-sa]
}
