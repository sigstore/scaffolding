// KMS resources
module "kms" {
  source = "../kms"

  region       = var.region
  project_id   = var.project_id
  cluster_name = var.cluster_name

  name     = var.kms_keyring
  location = var.kms_location
  key_name = var.kms_key_name
}

// Redis for Rekor.
module "redis" {
  source = "../redis"

  region     = var.region
  project_id = var.project_id

  cluster_name = var.cluster_name

  network = var.network
}
