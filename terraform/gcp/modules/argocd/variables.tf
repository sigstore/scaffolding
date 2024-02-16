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

variable "argocd_chart_version" {
  description = "Version of ArgoCD Helm chart. Versions listed here https://artifacthub.io/packages/helm/argo/argo-cd"
  type        = string
}

variable "argocd_apps_chart_version" {
  description = "Version of ArgoCD-Apps Helm chart. Versions listed here https://artifacthub.io/packages/helm/argo/argocd-apps"
  type        = string
}

variable "argo_chart_values_yaml_path" {
  description = "Path to ArgoCD Helm chart value yaml."
  type        = string
}

variable "argo_apps_chart_values_yaml_path" {
  description = "Path to ArgoCD-Apps Helm chart value yaml."
  type        = string
}

variable "github_repo" {
  description = "Github repo for running Github Actions from."
  type        = string
}

variable "gcp_secret_name_ssh" {
  description = "GCP Secret name that holds the SSH key for GitHub repository access."
  type        = string
}

variable "gcp_secret_name_slack_token" {
  description = "GCP Secret name that holds the slack token to argocd send notifications."
  type        = string
}

variable "gcp_secret_name_argocd_oauth_client_id" {
  description = "GCP Secret name that holds the OAuth client ID used by ArgoCD's Dex instance."
  type        = string
}

variable "gcp_secret_name_argocd_oauth_client_secret" {
  description = "GCP Secret name that holds the OAuth client secret used by ArgoCD's Dex instance."
  type        = string
}
