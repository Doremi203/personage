data "yandex_vpc_security_group" "default" {
  security_group_id = "enpopr5ic3p9afs56600"
}

resource "yandex_logging_group" "backend_logging_group" {
  name             = "backend"
  retention_period = "259200s"
}

module "tasker" {
  source = "./modules/service-instance-group"

  service_name          = "tasker"
  container_registry_id = yandex_container_registry.container_registry.id
  docker_compose_file   = "${path.module}/pods-specs/docker-compose.tasker.yml"
  domain                = var.domain

  dns_zone_id    = yandex_dns_zone.main_domain_public_zone.id
  http_router_id = yandex_alb_http_router.https_router.id
  alb_ip_address = yandex_alb_load_balancer.main_alb.listener[0].endpoint[0].address[0].external_ipv4_address[0].address

  network_id                = data.yandex_vpc_network.default.network_id
  folder_id                 = data.yandex_vpc_network.default.folder_id
  subnet_id                 = yandex_vpc_subnet.services_central1_d.id
  alb_security_group_id     = yandex_vpc_security_group.alb.id
  postgres_security_group_id = yandex_vpc_security_group.postgres_db.id
  bastion_security_group_id = data.yandex_vpc_security_group.default.id

  instance_count         = 2
  instance_memory        = 1
  instance_cores         = 2
  instance_core_fraction = 20
  instance_disk_size     = 15
}

module "notificator" {
  source = "./modules/service-instance-group"

  service_name          = "notificator"
  container_registry_id = yandex_container_registry.container_registry.id
  docker_compose_file   = "${path.module}/pods-specs/docker-compose.notificator.yml"
  domain                = var.domain

  dns_zone_id    = yandex_dns_zone.main_domain_public_zone.id
  http_router_id = yandex_alb_http_router.https_router.id
  alb_ip_address = yandex_alb_load_balancer.main_alb.listener[0].endpoint[0].address[0].external_ipv4_address[0].address

  network_id                = data.yandex_vpc_network.default.network_id
  folder_id                 = data.yandex_vpc_network.default.folder_id
  subnet_id                 = yandex_vpc_subnet.services_central1_d.id
  alb_security_group_id     = yandex_vpc_security_group.alb.id
  postgres_security_group_id = yandex_vpc_security_group.postgres_db.id
  bastion_security_group_id = data.yandex_vpc_security_group.default.id

  instance_count         = 2
  instance_memory        = 1
  instance_cores         = 2
  instance_core_fraction = 20
  instance_disk_size     = 15
}

module "auth" {
  source = "./modules/service-instance-group"

  service_name          = "auth"
  container_registry_id = yandex_container_registry.container_registry.id
  docker_compose_file   = "${path.module}/pods-specs/docker-compose.auth.yml"
  domain                = var.domain

  dns_zone_id    = yandex_dns_zone.main_domain_public_zone.id
  http_router_id = yandex_alb_http_router.https_router.id
  alb_ip_address = yandex_alb_load_balancer.main_alb.listener[0].endpoint[0].address[0].external_ipv4_address[0].address

  network_id                = data.yandex_vpc_network.default.network_id
  folder_id                 = data.yandex_vpc_network.default.folder_id
  subnet_id                 = yandex_vpc_subnet.services_central1_d.id
  alb_security_group_id     = yandex_vpc_security_group.alb.id
  postgres_security_group_id = yandex_vpc_security_group.postgres_db.id
  bastion_security_group_id = data.yandex_vpc_security_group.default.id

  instance_count         = 2
  instance_memory        = 1
  instance_cores         = 2
  instance_core_fraction = 20
  instance_disk_size     = 15

  liveliness_check = {
    healthy_threshold   = 2
    unhealthy_threshold = 3
    interval            = 2
    timeout             = 1
    http_options = {
      path = "/liveness"
      port = 80
    }
  }
}
