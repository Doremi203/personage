resource "yandex_mdb_redis_cluster_v2" "valkey" {
  name               = "main"
  environment        = "PRODUCTION"
  network_id         = data.yandex_vpc_network.default.network_id
  security_group_ids = [yandex_vpc_security_group.valkey.id]
  auth_sentinel      = true
  announce_hostnames = true
  persistence_mode   = "ON"
  config = {
    version  = "9.0-valkey"
    password = local.valkey_secret_map["password"]
  }
  resources = {
    resource_preset_id = "b3-c1-m4"
    disk_size          = 16
    disk_type_id       = "network-ssd"
  }
  hosts = {
    "host1" = {
      zone             = "ru-central1-d"
      subnet_id        = data.yandex_vpc_subnet.default-d.id
      replica_priority = 1
    }
  }

  maintenance_window = {
    type = "WEEKLY"
    day  = "SAT"
    hour = 3
  }

  access = {
    web_sql = true
  }
}

resource "yandex_vpc_security_group" "valkey" {
  name       = "valkey"
  network_id = data.yandex_vpc_network.default.network_id
}

resource "yandex_vpc_security_group_rule" "valkey_ingress_redis" {
  security_group_binding = yandex_vpc_security_group.valkey.id
  direction              = "ingress"
  description            = "Принимает трафик от telegram-auth сервиса для подключения напрямую к хостам"
  port                   = 6379
  protocol               = "TCP"
  security_group_id      = module.telegram-auth.security_group_id
}

resource "yandex_vpc_security_group_rule" "valkey_ingress_sentinel" {
  security_group_binding = yandex_vpc_security_group.valkey.id
  direction              = "ingress"
  description            = "Принимает трафик от telegram-auth сервиса для подключения через Sentinel"
  port                   = 26379
  protocol               = "TCP"
  security_group_id      = module.telegram-auth.security_group_id
}

data "yandex_lockbox_secret_version" "valkey" {
  secret_id  = "e6qjundo7djfkiccb0s0"
  version_id = "e6qjsqoe8h8fj8er2toi"
}

locals {
  valkey_secret_map = {
    for e in data.yandex_lockbox_secret_version.valkey.entries :
    e.key => e.text_value
  }
}
