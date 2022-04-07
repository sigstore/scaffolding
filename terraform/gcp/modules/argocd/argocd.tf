
// ArgoCD
resource "kubernetes_namespace_v1" "argocd" {
  metadata {
    name = "argocd"
  }
}

resource "kubectl_manifest" "externalsecret_argocd_ssh" {
  yaml_body = <<YAML
apiVersion: external-secrets.io/v1alpha1
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

resource "helm_release" "argocd" {
  name       = "argocd"
  namespace  = "argocd"
  chart      = "argo-cd"
  repository = "https://argoproj.github.io/argo-helm"
  version    = var.argocd_chart_version

  values = [
    file(var.argo_chart_values_yaml_path)
  ]

  depends_on = [
    kubectl_manifest.externalsecret_argocd_ssh
  ]
}
