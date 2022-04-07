output "network_name" {
  value = google_compute_network.network.name
}

output "network_self_link" {
  value = google_compute_network.network.self_link
}

output "subnetwork_self_link" {
  value = google_compute_subnetwork.subnetwork.self_link
}

output "secondary_ip_range" {
  value = google_compute_subnetwork.subnetwork.secondary_ip_range
}
