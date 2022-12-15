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
    kubectl_manifest.externalsecret_argocd_slack
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
