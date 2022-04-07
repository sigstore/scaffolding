provider "google" {
  project = var.project
}

provider "google-beta" {
  project = var.project
}

module "bastion" {
  source = "./../../"

  region     = var.region
  network    = "default"
  subnetwork = "default"

  kubernetes_api_address = "1.1.1.1"
}

output "connect-cmd" {
  value = module.bastion.ssh-cmd
}
