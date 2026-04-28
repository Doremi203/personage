resource "yandex_storage_bucket" "frontend_s3" {
  bucket = var.domain

  website {
    index_document = "index.html"
    error_document = "index.html"
  }

  https {
    certificate_id = yandex_cm_certificate.main_domain_cert.id
  }
}

resource "yandex_storage_bucket_grant" "public_read" {
  bucket = yandex_storage_bucket.frontend_s3.bucket
  acl    = "public-read"
}

resource "yandex_cdn_origin_group" "frontend_origin_group" {
  name     = "personamage-origin-group"
  use_next = true
  origin {
    source = "${var.domain}.website.yandexcloud.net"
  }
}

resource "yandex_cdn_resource" "frontend_cdn" {
  cname             = var.domain
  active            = true
  origin_protocol   = "http"
  origin_group_name = yandex_cdn_origin_group.frontend_origin_group.name
  options {
    custom_host_header     = "${var.domain}.website.yandexcloud.net"
    redirect_http_to_https = true
    static_response_headers = {
      is-cdn = "yes"
    }
  }

  ssl_certificate {
    type                   = "certificate_manager"
    certificate_manager_id = yandex_cm_certificate.main_domain_cert.id
  }
}

resource "yandex_dns_recordset" "frontend_apex_record" {
  zone_id = yandex_dns_zone.main_domain_public_zone.id
  name    = "${var.domain}."
  type    = "ANAME"
  ttl     = 600
  # Hotfix: bypass the Yandex CDN and serve directly from the S3 website
  # endpoint. The CDN was caching sw.js / index.html for ~24h regardless of
  # the origin's Cache-Control headers and `yc cdn cache purge` was failing
  # to actually evict edge caches, which broke PWA service-worker installs
  # (assets referenced by the cached sw.js had been deleted by deploys).
  # Re-point at `yandex_cdn_resource.frontend_cdn.provider_cname` once the
  # CDN issue is resolved.
  data = ["${var.domain}.website.yandexcloud.net"]
}

moved {
  from = yandex_dns_recordset.frontend_s3_record
  to   = yandex_dns_recordset.frontend_apex_record
}
