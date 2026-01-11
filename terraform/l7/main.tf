terraform {
  required_providers {
    yandex = {
      source = "yandex-cloud/yandex"
    }
  }
  required_version = ">= 0.13"
}

provider "yandex" {
  zone = "ru-central1-d"
}

data "yandex_vpc_network" "default" {
  name = "default"
}

data "yandex_vpc_subnet" "default-d" {
  subnet_id = "fl8phmfmikiae99hlidi"
}

/*resource "yandex_alb_load_balancer" "main_alb" {
  name = "main"
  network_id = data.yandex_vpc_network.default.network_id

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
        external_ipv4_address {}
      }
      ports = [443]
    }
  }
}*/