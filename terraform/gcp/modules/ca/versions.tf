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
  }
}
