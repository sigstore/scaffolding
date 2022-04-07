output "ssh_cmd" {
  description = "Instructions to connect to bastion"
  value       = "gcloud compute ssh --zone ${google_compute_instance.bastion.zone} ${google_compute_instance.bastion.name} --tunnel-through-iap --project ${google_compute_instance.bastion.project}"
}

output "name" {
  value = google_compute_instance.bastion.name
}

output "zone" {
  value = google_compute_instance.bastion.zone
}

output "project" {
  value = google_compute_instance.bastion.project
}

output "ip_address" {
  description = "private IP address of bastion"
  value       = google_compute_instance.bastion.network_interface.0.network_ip
}
