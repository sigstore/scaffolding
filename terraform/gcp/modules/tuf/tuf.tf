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
    "storage.googleapis.com", // For GCS bucket. roles/storage.admin
  ])
  project = var.project_id
  service = each.key

  // Do not disable the service on destroy. On destroy, we are going to
  // destroy the project, but we need the APIs available to destroy the
  // underlying resources.
  disable_on_destroy = false
}

resource "google_storage_bucket" "tuf" {
  name     = var.tuf_bucket
  location = var.region
  project  = var.project_id

  storage_class               = var.storage_class
  uniform_bucket_level_access = true

  versioning {
    enabled = true
  }

  lifecycle_rule {
    action {
      type = "Delete"
    }
    condition {
      with_state         = "ANY"
      num_newer_versions = 10
    }
  }
  lifecycle_rule {
    action {
      type = "Delete"
    }
    condition {
      days_since_noncurrent_time = 730
    }
  }

  dynamic "logging" {
    for_each = var.gcs_logging_enabled ? [1] : []
    content {
      log_bucket = var.gcs_logging_bucket
    }
  }
}

resource "google_storage_bucket_iam_member" "public_tuf_member" {
  bucket = google_storage_bucket.tuf.name
  role   = "roles/storage.legacyObjectReader"
  member = var.tuf_bucket_member

  depends_on = [google_storage_bucket.tuf]
}

resource "google_storage_bucket_iam_member" "tuf_sa_editor" {
  for_each = toset([
    "roles/storage.objectUser",
    "roles/storage.legacyBucketReader"
  ])

  bucket = google_storage_bucket.tuf.name
  role   = each.key
  member = format("serviceAccount:%s@%s.iam.gserviceaccount.com", var.tuf_service_account_name, var.project_id)

  depends_on = [google_storage_bucket.tuf, google_service_account.tuf-sa]
}

resource "google_storage_bucket" "tuf_preprod" {
  name     = var.tuf_preprod_bucket
  location = var.region
  project  = var.project_id

  storage_class               = var.storage_class
  uniform_bucket_level_access = true

  versioning {
    enabled = true
  }

  lifecycle_rule {
    action {
      type = "Delete"
    }
    condition {
      with_state         = "ANY"
      num_newer_versions = 10
    }
  }
  lifecycle_rule {
    action {
      type = "Delete"
    }
    condition {
      days_since_noncurrent_time = 730
    }
  }

  dynamic "logging" {
    for_each = var.gcs_logging_enabled ? [1] : []
    content {
      log_bucket = var.gcs_logging_bucket
    }
  }
}

resource "google_storage_bucket_iam_member" "public_tuf_preprod_member" {
  bucket = google_storage_bucket.tuf_preprod.name
  role   = "roles/storage.legacyObjectReader"
  member = var.tuf_bucket_member

  depends_on = [google_storage_bucket.tuf_preprod]
}

resource "google_storage_bucket_iam_member" "tuf_sa_preprod_editor" {
  for_each = toset([
    "roles/storage.objectUser",
    "roles/storage.legacyBucketReader"
  ])

  bucket = google_storage_bucket.tuf_preprod.name
  role   = each.key
  member = format("serviceAccount:%s@%s.iam.gserviceaccount.com", var.tuf_service_account_name, var.project_id)

  depends_on = [google_storage_bucket.tuf_preprod, google_service_account.tuf-sa]
}
