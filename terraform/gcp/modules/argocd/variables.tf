variable "argocd_chart_version" {
  description = "Version of ArgoCD Helm chart. Versions listed here https://artifacthub.io/packages/helm/argo/argo-cd"
  type        = string
}

variable "argo_chart_values_yaml_path" {
  description = "Path to ArgoCD Helm chart value yaml."
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
