// Resources for Certificate Authority
module "ca" {
  source = "../ca"

  region       = var.region
  project_id   = var.project_id
  ca_pool_name = var.ca_pool_name
}
