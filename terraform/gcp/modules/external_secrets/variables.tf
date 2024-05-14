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

variable "external_secrets_chart_version" {
  description = "Version of External-Secrets Helm chart. Versions listed here https://artifacthub.io/packages/helm/external-secrets-operator/external-secrets"
  type        = string
}

variable "external_secrets_chart_values_yaml_path" {
  description = "Path to External Secrets Helm Chart values YAML."
  type        = string
}

variable "project_id" {
  type = string
  validation {
    condition     = length(var.project_id) > 0
    error_message = "Must specify project_id variable."
  }
}

variable "mysql_dbname" {
  type        = string
  description = "Name of MySQL database."
  default     = "trillian"
}

variable "rekor_mysql_dbname" {
  type        = string
  description = "Name of the MySQL database for Rekor search indexes."
  default     = "searchindexes"
}
