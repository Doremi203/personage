resource "yandex_vpc_subnet" "services_central1_d" {
  name           = "services-central1-d"
  network_id     = data.yandex_vpc_network.default.id
  zone           = "ru-central1-d"
  v4_cidr_blocks = ["10.131.0.0/24"]
  route_table_id = yandex_vpc_route_table.services_route_table.id
}

resource "yandex_vpc_gateway" "services_nat_gateway" {
  name = "services-gateway"
  shared_egress_gateway {}
}

resource "yandex_vpc_route_table" "services_route_table" {
  name       = "services-route-table"
  network_id = data.yandex_vpc_network.default.id

  static_route {
    destination_prefix = "0.0.0.0/0"
    gateway_id         = yandex_vpc_gateway.services_nat_gateway.id
  }
}
