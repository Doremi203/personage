resource "yandex_iam_service_account" "docker_images_pusher" {
  name = "docker-images-pusher"
}