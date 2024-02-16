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
    "admin.googleapis.com",         // For accessing Directory API
    "secretmanager.googleapis.com", // For Secrets
  ])
  project = var.project_id
  service = each.key

  // Do not disable the service on destroy. On destroy, we are going to
  // destroy the project, but we need the APIs available to destroy the
  // underlying resources.
  disable_on_destroy = false
}

// ArgoCD
resource "kubernetes_namespace_v1" "argocd" {
  metadata {
    name = "argocd"
  }
}

resource "kubectl_manifest" "externalsecret_argocd_ssh" {
  yaml_body = <<YAML
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: gcp-external-secret-argocd-ssh
  namespace: "${kubernetes_namespace_v1.argocd.metadata[0].name}"
spec:
  secretStoreRef:
    kind: ClusterSecretStore
    name: gcp-backend
  target:
    name: argocd-repository-credentials
    template:
      metadata:
        labels:
          argocd.argoproj.io/secret-type: repository
  data:
  - secretKey: sshPrivateKey
    remoteRef:
      key: "${var.gcp_secret_name_ssh}"
YAML

  depends_on = [
    kubernetes_namespace_v1.argocd
  ]
}

resource "kubectl_manifest" "externalsecret_argocd_slack" {
  yaml_body = <<YAML
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: slack-argocd-notification
  namespace: "${kubernetes_namespace_v1.argocd.metadata[0].name}"
spec:
  secretStoreRef:
    kind: ClusterSecretStore
    name: gcp-backend
  target:
    name: argocd-notifications-secret
  data:
  - secretKey: slack-token
    remoteRef:
      key: "${var.gcp_secret_name_slack_token}"
YAML

  depends_on = [
    kubernetes_namespace_v1.argocd
  ]
}

resource "helm_release" "argocd" {
  name       = "argocd"
  namespace  = "argocd"
  chart      = "argo-cd"
  repository = "https://argoproj.github.io/argo-helm"
  version    = var.argocd_chart_version
  timeout    = 900

  values = [
    file(var.argo_chart_values_yaml_path)
  ]

  depends_on = [
    kubectl_manifest.externalsecret_argocd_ssh,
    kubectl_manifest.externalsecret_argocd_slack,
    kubectl_manifest.externalsecret_argocd_oauth_client_credentials
  ]
}

resource "helm_release" "argocd_apps" {
  name       = "argocd-apps"
  namespace  = "argocd"
  chart      = "argocd-apps"
  repository = "https://argoproj.github.io/argo-helm"
  version    = var.argocd_apps_chart_version
  timeout    = 900

  values = [
    file(var.argo_apps_chart_values_yaml_path)
  ]

  depends_on = [
    helm_release.argocd
  ]
}

# this needs to be manually granted access via admin console
# - Client ID needs to be enabled for domain-wide delegation
# - SA needs to be granted "Groups Reader" role
resource "google_service_account" "argocd-directory-api-sa" {
  account_id   = "argocd-directory-api-sa"
  display_name = "ArgoCD Directory API Service Account"
  project      = var.project_id
}

resource "kubectl_manifest" "externalsecret_argocd_oauth_client_credentials" {
  yaml_body = <<YAML
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: gcp-external-secret-argocd-oauth-client-credentials
  namespace: "${kubernetes_namespace_v1.argocd.metadata[0].name}"
spec:
  secretStoreRef:
    kind: ClusterSecretStore
    name: gcp-backend
  target:
    name: argocd-oauth-client-credentials
  data:
  - secretKey: client-id
    remoteRef:
      key: "${google_secret_manager_secret.argocd-oauth-client-id.secret_id}"
  - secretKey: client-secret
    remoteRef:
      key: "${google_secret_manager_secret.argocd-oauth-client-secret.secret_id}"
YAML

  depends_on = [
    kubernetes_namespace_v1.argocd
  ]
}

resource "google_secret_manager_secret" "argocd-oauth-client-id" {
  secret_id = var.gcp_secret_name_argocd_oauth_client_id
  project   = var.project_id

  replication {
    auto {}
  }
  depends_on = [google_project_service.service]
}

resource "google_secret_manager_secret" "argocd-oauth-client-secret" {
  secret_id = var.gcp_secret_name_argocd_oauth_client_secret
  project   = var.project_id

  replication {
    auto {}
  }
  depends_on = [google_project_service.service]
}
