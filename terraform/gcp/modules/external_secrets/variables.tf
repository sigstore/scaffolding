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
