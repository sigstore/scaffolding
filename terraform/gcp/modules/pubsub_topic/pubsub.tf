/**
 * Copyright 2023 The Sigstore Authors
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

resource "google_pubsub_topic" "topic" {
  name    = var.pubsub_topic_name
  project = var.project_id
}

data "google_iam_policy" "topic_iam" {
  binding {
    role    = "roles/pubsub.publisher"
    members = ["serviceAccount:${var.publisher_sa_email}"]
  }
  binding {
    role    = "roles/pubsub.subscriber"
    members = var.pubsub_topic_consumers
  }
}

resource "google_pubsub_topic_iam_policy" "topic_iam" {
  project     = google_pubsub_topic.topic.project
  topic       = google_pubsub_topic.topic.name
  policy_data = data.google_iam_policy.topic_iam
}
