resource "yandex_dns_zone" "main_domain_public_zone" {
  name = "public-zone"
  zone = "persomanage.ru."
  public = true
}
