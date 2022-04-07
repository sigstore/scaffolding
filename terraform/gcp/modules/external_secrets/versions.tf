terraform {
  required_version = ">= 1.1.3, < 1.2.0"

  required_providers {
    google = {
      version = ">= 4.11.0, < 4.12.0"
      source  = "hashicorp/google-beta"
    }
    google-beta = {
      version = ">= 4.11.0, < 4.12.0"
      source  = "hashicorp/google-beta"
    }
    random = {
      version = ">= 3.1.0, < 3.2.0"
      source  = "hashicorp/random"
    }
    kubectl = {
      source  = "gavinbunney/kubectl"
      version = "1.13.1"
    }
    helm = {
      // Using custom built version for proxy access.
      // Switch to public instance once https://github.com/hashicorp/terraform-provider-helm/pull/834 lands.
      source  = "nsmith5/helm"
      version = "2.4.3"
    }
  }
}
