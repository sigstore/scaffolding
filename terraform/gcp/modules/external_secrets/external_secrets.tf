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


locals {
  namespace = "external-secrets"
  k8s_sa    = "external-secrets"
}

// External-Secrets
resource "helm_release" "external_secrets" {
  name             = "external-secrets"
  namespace        = local.namespace
  create_namespace = true
  chart            = "external-secrets"
  repository       = "https://charts.external-secrets.io"
  version          = var.external_secrets_chart_version

  values = [
    file(var.external_secrets_chart_values_yaml_path)
  ]
}

resource "google_service_account" "external_secrets_sa" {
  account_id   = "external-secrets-sa"
  display_name = "external-secrets Service Account"
  project      = var.project_id
}

resource "google_project_iam_member" "external_secrets_binding" {
  project    = var.project_id
  role       = "roles/secretmanager.secretAccessor"
  member     = "serviceAccount:${google_service_account.external_secrets_sa.email}"
  depends_on = [google_service_account.external_secrets_sa]
}

resource "google_service_account_iam_member" "gke_sa_iam_member_external_secrets" {
  service_account_id = google_service_account.external_secrets_sa.name
  role               = "roles/iam.workloadIdentityUser"
  member             = "serviceAccount:${var.project_id}.svc.id.goog[${local.namespace}/${local.k8s_sa}]"
  depends_on         = [google_service_account.external_secrets_sa]
}

// Needs roles/iam.serviceAccountKeyAdmin
resource "google_service_account_key" "external_secrets_key" {
  service_account_id = google_service_account.external_secrets_sa.name
}

resource "kubectl_manifest" "secretstore_gcp_backend" {
  yaml_body = <<YAML
apiVersion: external-secrets.io/v1beta1
kind: ClusterSecretStore
metadata:
  name: gcp-backend
spec:
  provider:
      gcpsm:
        projectID: "${var.project_id}"
YAML

  depends_on = [helm_release.external_secrets]
}

resource "kubectl_manifest" "trillian_namespace" {
  yaml_body = <<YAML
apiVersion: v1
kind: Namespace
metadata:
  name: trillian-system
YAML
}

resource "kubectl_manifest" "trillian_mysql_external_secret" {
  yaml_body = <<YAML
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: trillian-mysql
  namespace: trillian-system
spec:
  secretStoreRef:
    kind: ClusterSecretStore
    name: gcp-backend
  target:
    name: trillian-mysql
    template:
      data:
        mysql-database: "${var.mysql_dbname}"
        mysql-password: "{{ .mysqlPassword | toString }}"  # <-- convert []byte to string
        mysql-user: trillian
  data:
  - secretKey: mysqlPassword
    remoteRef:
      key: mysql-password
YAML

  depends_on = [
    kubectl_manifest.secretstore_gcp_backend,
    kubectl_manifest.trillian_namespace
  ]
}

resource "kubectl_manifest" "rekor_namespace" {
  yaml_body = <<YAML
apiVersion: v1
kind: Namespace
metadata:
  name: rekor-system
YAML
}

resource "kubectl_manifest" "rekor_mysql_external_secret" {
  yaml_body = <<YAML
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: rekor-mysql
  namespace: rekor-system
spec:
  secretStoreRef:
    kind: ClusterSecretStore
    name: gcp-backend
  target:
    name: rekor-mysql
    template:
      data:
        mysql-database: "${var.rekor_mysql_dbname}"
        mysql-password: "{{ .mysqlPassword | toString }}"  # <-- convert []byte to string
        mysql-user: trillian
  data:
  - secretKey: mysqlPassword
    remoteRef:
      key: mysql-password
YAML

  depends_on = [
    kubectl_manifest.secretstore_gcp_backend,
    kubectl_manifest.rekor_namespace
  ]
}
