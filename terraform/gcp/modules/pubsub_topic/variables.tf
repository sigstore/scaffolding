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

variable "pubsub_topic_name" {
  description = "Name of the pubsub topic"
  type        = string
}

variable "pubsub_topic_consumers" {
  description = "IAM members that can consume messages from the topic"
  type        = list(string)
}

variable "publisher_sa_email" {
  description = "The service account that can publish meessages to the topic"
  type        = string
}

variable "project_id" {
  description = "The project to create the pubsub resources in"
  type        = string
}
