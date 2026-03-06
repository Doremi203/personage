data "yandex_vpc_network" "default" {
  name = "default"
}

data "yandex_vpc_subnet" "default-d" {
  subnet_id = "fl8phmfmikiae99hlidi"
}

resource "yandex_vpc_security_group" "alb" {
  name       = "alb"
  network_id = data.yandex_vpc_network.default.network_id

  ingress {
    protocol       = "TCP"
    description    = "Принимает пользовательский трафик на 80 порте"
    port           = 80
    v4_cidr_blocks = ["0.0.0.0/0"]
  }
  ingress {
    protocol       = "TCP"
    description    = "Принимает пользовательский трафик на 443 порте"
    port           = 443
    v4_cidr_blocks = ["0.0.0.0/0"]
  }
  ingress {
    protocol          = "TCP"
    description       = "Получение входящего трафика для проверки состояния узлов балансировщика в разных зонах доступности"
    port              = 30080
    predefined_target = "loadbalancer_healthchecks"
  }

  egress {
    protocol       = "ANY"
    description    = "any"
    v4_cidr_blocks = ["0.0.0.0/0"]
    from_port      = 0
    to_port        = 65535
  }
}

resource "yandex_vpc_address" "public_alb_static_address" {
  name = "public-alb-static-address"
  external_ipv4_address {
    zone_id = "ru-central1-d"
  }
}

resource "yandex_alb_load_balancer" "main_alb" {
  name               = "main"
  network_id         = data.yandex_vpc_network.default.network_id
  security_group_ids = [yandex_vpc_security_group.alb.id]

  allocation_policy {
    location {
      subnet_id = data.yandex_vpc_subnet.default-d.subnet_id
      zone_id   = "ru-central1-d"
    }
  }

  listener {
    name = "https-listener"
    endpoint {
      address {
        external_ipv4_address {
          address = yandex_vpc_address.public_alb_static_address.external_ipv4_address[0].address
        }
      }
      ports = [443]
    }
    tls {
      default_handler {
        certificate_ids = [yandex_cm_certificate.main_domain_cert.id]
        http_handler {
          http_router_id = yandex_alb_http_router.https_router.id
        }
      }
    }
  }
  listener {
    name = "http-listener"
    endpoint {
      address {
        external_ipv4_address {
          address = yandex_vpc_address.public_alb_static_address.external_ipv4_address[0].address
        }
      }
      ports = [80]
    }
    http {
      redirects {
        http_to_https = true
      }
    }
  }
}

resource "yandex_alb_http_router" "https_router" {
  name = "https-router"
}
