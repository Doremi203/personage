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
}

resource "yandex_vpc_address" "public_alb_static_address" {
  name       = "public-alb-static-address"
  external_ipv4_address {
    zone_id = "ru-central1-d"
  }
}

resource "yandex_dns_recordset" "alb-record" {
  zone_id = yandex_dns_zone.main_domain_public_zone.id
  name    = "${var.domain}."
  ttl     = 600
  type    = "A"
  data    = [yandex_alb_load_balancer.main_alb.listener[0].endpoint[0].address[0].external_ipv4_address[0].address]
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
      sni_handler {
        name = "main-sni"
        server_names = [var.domain]
        handler {
          http_handler {
            http_router_id = yandex_alb_http_router.https_router.id
          }
          certificate_ids = [yandex_cm_certificate.main_domain_cert.id]
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

resource "yandex_alb_virtual_host" "https_main_vhost" {
  name           = "https-main"
  http_router_id = yandex_alb_http_router.https_router.id
  authority      = [var.domain]
  route {
    name = "main-route"
    http_route {
      direct_response_action {
        status = 200
        body   = "Persomanage main server"
      }
    }
  }
}
