resource "yandex_dns_zone" "main_domain_public_zone" {
  name   = "public-zone"
  zone   = "persomanage.ru."
  public = true
}

resource "yandex_dns_recordset" "google_oauth" {
  name    = "${var.domain}."
  ttl     = 600
  type    = "TXT"
  zone_id = yandex_dns_zone.main_domain_public_zone.id
  data    = ["google-site-verification=Buyg9peKKJlc91ooDa77KmI_BU5V186_GKe2XZhgcZw"]
}
