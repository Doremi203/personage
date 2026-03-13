terraform {
  required_providers {
    yandex = {
      source  = "yandex-cloud/yandex"
      version = "0.178.0"
    }
  }
}

locals {
  # Collect all unique health check ports
  health_check_ports = toset([
    var.liveliness_check.http_options.port,
    var.health_check.healthcheck_port
  ])
}

resource "yandex_container_repository" "service_repository" {
  name = "${var.container_registry_id}/${var.service_name}"
}

resource "yandex_container_repository_lifecycle_policy" "service" {
  repository_id = yandex_container_repository.service_repository.id
  status        = "active"

  rule {
    description = "delete images older than 24 hours"
    expire_period = "24h"
    tag_regexp   = ".*"
    untagged = true
    retained_top = 6
  }
}

resource "yandex_iam_service_account" "service" {
  name = var.service_name
}

resource "yandex_container_registry_iam_binding" "images_puller_iam_binding" {
  registry_id = var.container_registry_id
  role        = "container-registry.images.puller"

  members = [
    "serviceAccount:${yandex_iam_service_account.service.id}",
  ]
}

resource "yandex_resourcemanager_folder_iam_member" "service_logging_writer" {
  folder_id = var.folder_id
  role      = "logging.writer"
  member    = "serviceAccount:${yandex_iam_service_account.service.id}"
}

resource "yandex_resourcemanager_folder_iam_member" "service_vpc_user" {
  folder_id = var.folder_id
  role      = "vpc.user"
  member    = "serviceAccount:${yandex_iam_service_account.service.id}"
}

resource "yandex_resourcemanager_folder_iam_member" "service_lockbox_viewer" {
  folder_id = var.folder_id
  role      = "lockbox.payloadViewer"
  member    = "serviceAccount:${yandex_iam_service_account.service.id}"
}

resource "yandex_resourcemanager_folder_iam_member" "kms_encrypter_decrypter" {
  folder_id = var.folder_id
  role      = "kms.keys.encrypterDecrypter"
  member    = "serviceAccount:${yandex_iam_service_account.service.id}"
}

resource "yandex_resourcemanager_folder_iam_member" "service_compute_editor" {
  folder_id = var.folder_id
  role      = "compute.editor"
  member    = "serviceAccount:${yandex_iam_service_account.service.id}"
}

resource "yandex_resourcemanager_folder_iam_member" "service_account_user" {
  folder_id = var.folder_id
  role      = "iam.serviceAccounts.user"
  member    = "serviceAccount:${yandex_iam_service_account.service.id}"
}

resource "yandex_resourcemanager_folder_iam_member" "service_alb_editor" {
  folder_id = var.folder_id
  role      = "alb.editor"
  member    = "serviceAccount:${yandex_iam_service_account.service.id}"
}

resource "yandex_vpc_security_group" "service" {
  name       = var.service_name
  network_id = var.network_id

  ingress {
    protocol          = "TCP"
    description       = "SSH access"
    security_group_id = var.bastion_security_group_id
    port              = 22
  }
  ingress {
    protocol          = "TCP"
    description       = "HTTP from ALB"
    security_group_id = var.alb_security_group_id
    port              = var.http_port
  }
  ingress {
    protocol          = "TCP"
    description       = "GRPC from ALB"
    security_group_id = var.alb_security_group_id
    port              = var.grpc_port
  }

  # Dynamic ingress rules for health check ports
  dynamic "ingress" {
    for_each = local.health_check_ports
    content {
      protocol          = "TCP"
      description       = "Health checks from load balancer on port ${ingress.value}"
      port              = ingress.value
      predefined_target = "loadbalancer_healthchecks"
    }
  }

  egress {
    protocol       = "TCP"
    description    = "HTTPS egress for pulling images"
    v4_cidr_blocks = ["0.0.0.0/0"]
    port           = 443
  }
  egress {
    protocol = "TCP"
    description = "to Postgres"
    security_group_id = var.postgres_security_group_id
    port = 6432
  }
}

resource "yandex_dns_recordset" "service_alb_record" {
  zone_id = var.dns_zone_id
  name    = "${var.service_name}.${var.domain}."
  ttl     = 600
  type    = "A"
  data    = [var.alb_ip_address]
}

resource "yandex_compute_instance_group" "service" {
  name               = var.service_name
  service_account_id = yandex_iam_service_account.service.id

  depends_on = [
    yandex_resourcemanager_folder_iam_member.service_vpc_user,
    yandex_resourcemanager_folder_iam_member.service_compute_editor,
    yandex_resourcemanager_folder_iam_member.service_account_user,
    yandex_resourcemanager_folder_iam_member.service_alb_editor
  ]

  instance_template {
    platform_id = "standard-v3"

    resources {
      memory        = var.instance_memory
      cores         = var.instance_cores
      core_fraction = var.instance_core_fraction
    }

    scheduling_policy {
      preemptible = var.preemptible
    }

    boot_disk {
      mode = "READ_WRITE"
      initialize_params {
        image_id = "fd8f946rrpej0cptu18n"
        type     = "network-hdd"
        size     = var.instance_disk_size
      }
    }

    network_interface {
      network_id         = var.network_id
      subnet_ids         = [var.subnet_id]
      security_group_ids = [yandex_vpc_security_group.service.id]
    }

    metadata = {
      docker-compose = file(var.docker_compose_file)
      enable-oslogin = true
    }

    service_account_id = yandex_iam_service_account.service.id
  }

  scale_policy {
    fixed_scale {
      size = var.instance_count
    }
  }

  allocation_policy {
    zones = [var.instance_zone]
  }

  deploy_policy {
    max_unavailable = 1
    max_creating    = 2
    max_expansion   = 1
    max_deleting    = 2
  }

  health_check {
    healthy_threshold   = var.liveliness_check.healthy_threshold
    unhealthy_threshold = var.liveliness_check.unhealthy_threshold
    interval            = var.liveliness_check.interval
    timeout             = var.liveliness_check.timeout
    http_options {
      path = var.liveliness_check.http_options.path
      port = var.liveliness_check.http_options.port
    }
  }

  application_load_balancer {
    target_group_name = var.service_name
  }
}

resource "yandex_alb_virtual_host" "service_https" {
  name           = "https-${var.service_name}"
  http_router_id = var.http_router_id
  authority      = ["${var.service_name}.${var.domain}"]

  route {
    name = "grpc-route"
    grpc_route {
      grpc_route_action {
        backend_group_id = yandex_alb_backend_group.service_grpc.id
        max_timeout = var.backend_timeout
      }
    }
  }
  route {
    name = "main-route"
    http_route {
      http_route_action {
        backend_group_id = yandex_alb_backend_group.service_http.id
        timeout          = var.backend_timeout
      }
    }
  }
}

resource "yandex_alb_backend_group" "service_http" {
  name = "${var.service_name}-http"

  http_backend {
    name             = "${var.service_name}-http-backend"
    weight           = 1
    port             = var.http_port
    target_group_ids = [yandex_compute_instance_group.service.application_load_balancer[0].target_group_id]

    healthcheck {
      interval         = var.health_check.interval
      timeout          = var.health_check.timeout
      healthcheck_port = var.health_check.healthcheck_port
      http_healthcheck {
        path = var.health_check.http_healthcheck.path
      }
    }
  }
}

resource "yandex_alb_backend_group" "service_grpc" {
  name = "${var.service_name}-grpc"

  grpc_backend {
    name             = "${var.service_name}-grpc-backend"
    weight           = 1
    port             = var.grpc_port
    target_group_ids = [yandex_compute_instance_group.service.application_load_balancer[0].target_group_id]

    healthcheck {
      interval         = var.health_check.interval
      timeout          = var.health_check.timeout
      healthcheck_port = var.health_check.healthcheck_port
      http_healthcheck {
        path = var.health_check.http_healthcheck.path
      }
    }
  }
}