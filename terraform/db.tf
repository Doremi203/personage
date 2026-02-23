resource "yandex_mdb_postgresql_cluster_v2" "main_db" {
  name        = "main"
  environment = "PRODUCTION"
  network_id  = data.yandex_vpc_network.default.id
  security_group_ids = [yandex_vpc_security_group.postgres_db.id]

  config {
    version = "17"
    resources {
      disk_size          = 10
      disk_type_id       = "network-ssd"
      resource_preset_id = "c3-c2-m4"
    }
    access = {
      web_sql = true
    }
  }

  hosts = {
    "host1d" = {
      zone      = "ru-central1-d"
      subnet_id = data.yandex_vpc_subnet.default-d.id
      assign_public_ip = true
    }
  }

  maintenance_window = {
    type = "WEEKLY"
    day  = "SAT"
    hour = 3
  }
}

resource "yandex_mdb_postgresql_database" "tasker" {
  cluster_id = yandex_mdb_postgresql_cluster_v2.main_db.id
  name       = "tasker"
  owner      = yandex_mdb_postgresql_user.admin.name
  extension {
    name = "pgvector"
  }
}

resource "yandex_mdb_postgresql_user" "admin" {
  cluster_id = yandex_mdb_postgresql_cluster_v2.main_db.id
  name       = "main"
  generate_password = true
}

resource "yandex_vpc_security_group" "postgres_db" {
  name = "postgres-db"
  network_id = data.yandex_vpc_network.default.id

  ingress {
    protocol = "TCP"
    description = "Принимает трафик на 6432 порте из интернета"
    port = 6432
    v4_cidr_blocks = ["0.0.0.0/0"]
  }
}