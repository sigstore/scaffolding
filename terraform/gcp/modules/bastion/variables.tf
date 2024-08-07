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

variable "project_id" {
  type    = string
  default = ""
  validation {
    condition     = length(var.project_id) > 0
    error_message = "Must specify project_id variable."
  }
}

variable "region" {
  type        = string
  description = "GCP region"
  default     = "us-west1"
}

variable "zone" {
  type        = string
  description = "Zone (a random zone will be selected for the bastion if one is not set)"
  default     = ""
}

variable "network" {
  type        = string
  description = "VPC network to deploy bastion into"
  default     = "default"
}

variable "subnetwork" {
  type        = string
  description = "VPC subnetwork to deploy bastion into"
  default     = "default"
}

variable "tunnel_accessor_sa" {
  type        = list(string)
  description = "Email of group to give access to the tunnel to"
}
