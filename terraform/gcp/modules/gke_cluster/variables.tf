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

// GENERAL

variable "project_id" {
  type    = string
  default = ""
  validation {
    condition     = length(var.project_id) > 0
    error_message = "Must specify project_id variable."
  }
}

variable "region" {
  type        = string
  description = "GCP region"
  default     = "us-west1"
}

variable "labels" {
  type    = map(any)
  default = {}
}

variable "tags" {
  type    = list(any)
  default = []
}

// CLUSTER

variable "cluster_name" {
  description = "The name to give the new Kubernetes cluster."
  type        = string
  default     = "sigstore-staging"
}

variable "channel" {
  type        = string
  description = "Release channel, options RAPID, REGULAR, STABLE"
  default     = "STABLE"
}

variable "datapath_provider" {
  type    = string
  default = "ADVANCED_DATAPATH"
}

variable "timeouts_create" {
  type    = string
  default = "30m"
}

variable "timeouts_update" {
  type    = string
  default = "40m"
}

variable "networking_mode" {
  type    = string
  default = "VPC_NATIVE"
}

variable "network" {
  type        = string
  description = "VPC network to deploy bastion into"
  default     = "default"
}

variable "subnetwork" {
  type        = string
  description = "VPC subnetwork to deploy bastion into"
  default     = "default"
}


// google_compute_subnetwork.subnetwork.secondary_ip_range.0.range_name
variable "cluster_secondary_range_name" {}
// google_compute_subnetwork.subnetwork.secondary_ip_range.1.range_name
variable "services_secondary_range_name" {}

// master_authorized_networks_config
variable "display_name" {
  type    = string
  default = "bastion"
}
// module.bastion.ip_address
variable "bastion_ip_address" {}

// private_cluster_config
variable "enable_private_endpoint" {
  type    = string
  default = "true"
}

variable "enable_private_nodes" {
  type    = string
  default = "true"
}

variable "master_ipv4_cidr_block" {
  type    = string
  default = "172.16.0.16/28"
}

// network_policy
variable "network_policy_enabled" {
  type    = bool
  default = false
}

variable "network_policy_provider" {
  type    = string
  default = "PROVIDER_UNSPECIFIED"
}

variable "cluster_network_tag" {
  type    = string
  default = ""
}

variable "cluster_autoscaling_profile" {
  type    = string
  default = "BALANCED"
}

variable "cluster_autoscaling_enabled" {
  type    = bool
  default = true
}

variable "resource_limits_resource_cpu_min" {
  type    = number
  default = 1
}

variable "resource_limits_resource_cpu_max" {
  type    = number
  default = 4
}

variable "resource_limits_resource_mem_min" {
  type    = number
  default = 4
}

variable "resource_limits_resource_mem_max" {
  type    = number
  default = 16
}

// NODE POOL

variable "node_pool_name" {
  type    = string
  default = "sigstore-node-pool"
}

variable "autoscaling_min_node" {
  type    = number
  default = 1
}

variable "autoscaling_max_node" {
  type    = number
  default = 10
}

variable "initial_node_count" {
  type    = number
  default = 3
}

variable "node_config_machine_type" {
  type    = string
  default = "n2-standard-4"
}

variable "node_config_disk_type" {
  type    = string
  default = "pd-ssd"
}

variable "node_config_image_type" {
  type    = string
  default = "COS_CONTAINERD"
}

variable "enable_secure_boot" {
  type    = bool
  default = true
}

variable "workload_metadata_config_mode" {
  type    = string
  default = "GKE_METADATA"
}

variable "managed_prometheus" {
  type    = bool
  default = true
}

variable "monitoring_components" {
  type    = list(string)
  default = ["SYSTEM_COMPONENTS"]
}

variable "security_group" {
  description = "Name of security group used for Google Groups RBAC within GKE Cluster"
  type        = string
}
