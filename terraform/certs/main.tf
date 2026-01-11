terraform {
  required_providers {
    yandex = {
      source = "yandex-cloud/yandex"
    }
  }
  required_version = ">= 0.13"

  backend "s3" {
    endpoints = {
      s3 = "https://storage.yandexcloud.net"
    }
    bucket = "persomanage-terraform-state"
    region = "ru-central1"
    key = "certs/terraform.tfstate"

    skip_region_validation      = true
    skip_credentials_validation = true
    skip_requesting_account_id  = true
    skip_s3_checksum            = true
  }
}

provider "yandex" {
  zone = "ru-central1-d"
}

resource "yandex_cm_certificate" "main_domain_cert" {
  name = "main-domain-cert"
  domains = ["persomanage.ru", "*.persomanage.ru"]
  managed {
    challenge_type = "DNS_CNAME"
    challenge_count = 1
  }
}

resource "yandex_dns_recordset" "main_domain_cert_challenge" {
  count   = yandex_cm_certificate.main_domain_cert.managed[0].challenge_count
  zone_id = "dnsdfcsi6b8fh21hj1ik"
  name    = yandex_cm_certificate.main_domain_cert.challenges[count.index].dns_name
  type    = yandex_cm_certificate.main_domain_cert.challenges[count.index].dns_type
  data    = [yandex_cm_certificate.main_domain_cert.challenges[count.index].dns_value]
  ttl     = 60
}