output "instance_group_id" {
  description = "ID of the created instance group"
  value       = yandex_compute_instance_group.service.id
}

output "target_group_id" {
  description = "ID of the target group for ALB"
  value       = yandex_compute_instance_group.service.application_load_balancer[0].target_group_id
}

output "backend_group_id" {
  description = "ID of the ALB backend group"
  value       = yandex_alb_backend_group.service_http.id
}

output "virtual_host_id" {
  description = "ID of the ALB virtual host"
  value       = yandex_alb_virtual_host.service_https.id
}

output "dns_record_id" {
  description = "ID of the DNS record"
  value       = yandex_dns_recordset.service_alb_record.id
}

output "service_fqdn" {
  description = "Fully qualified domain name for the service"
  value       = "${var.service_name}.${var.domain}"
}

output "security_group_id" {
  description = "ID of the created security group"
  value       = yandex_vpc_security_group.service.id
}

output "service_account_id" {
  description = "ID of the service account (created or provided)"
  value       = yandex_iam_service_account.service.id
}

