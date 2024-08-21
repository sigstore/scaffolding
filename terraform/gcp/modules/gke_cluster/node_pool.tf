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

// Separately managed node pool

resource "google_container_node_pool" "cluster_nodes" {
  name     = var.node_pool_name
  location = var.region
  cluster  = google_container_cluster.cluster.name
  project  = var.project_id

  initial_node_count = var.initial_node_count

  // NB: updates to initial_node_count are ignored
  // because they recreate the entire node pool.
  // ref: https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/container_node_pool#initial_node_count
  lifecycle {
    ignore_changes = [
      initial_node_count
    ]
  }

  autoscaling {
    min_node_count = var.autoscaling_min_node
    max_node_count = var.autoscaling_max_node
  }

  management {
    auto_repair  = true
    auto_upgrade = true
  }

  upgrade_settings {
    max_surge       = 2
    max_unavailable = 1
  }

  node_config {
    # Machine n2d needed for confidential gke nodes
    machine_type = var.node_config_machine_type
    disk_type    = var.node_config_disk_type
    image_type   = var.node_config_image_type

    metadata = {
      disable-legacy-endpoints = true
    }
    tags = [local.cluster_network_tag]
    shielded_instance_config {
      enable_secure_boot = var.enable_secure_boot
    }
    # Google recommends custom service accounts that have cloud-platform scope and permissions granted via IAM Roles.
    service_account = google_service_account.gke-sa.email
    oauth_scopes    = ["https://www.googleapis.com/auth/cloud-platform"]

    kubelet_config {
      cpu_cfs_quota      = false
      pod_pids_limit     = 0
      cpu_manager_policy = "none"
    }

    // Protect node metadata and enable Workload Identity
    // for this node pool.  "SECURE" just protects the metadata.
    // "EXPOSE" or not set allows for cluster takeover.
    // "GKE_METADATA" specifies that each pod's requests to the metadata
    // API for credentials should be intercepted and given the specific
    // credentials for that pod only and not the node's.
    workload_metadata_config {
      mode = var.workload_metadata_config_mode
    }
  }

  timeouts {
    create = var.timeouts_create
    update = var.timeouts_update
  }

  depends_on = [google_container_cluster.cluster]
}
