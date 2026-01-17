resource "yandex_iam_service_account" "docker_images_pusher" {
  name = "docker-images-pusher"
}

resource "yandex_container_registry_iam_binding" "images_pusher_iam_binding" {
  registry_id = yandex_container_registry.container_registry.id
  role        = "container-registry.images.pusher"

  members = [
    "serviceAccount:${yandex_iam_service_account.docker_images_pusher.id}",
  ]
}

resource "yandex_container_registry" "container_registry" {
  name = "internal-registry"
}

resource "yandex_container_repository" "tasker_repository" {
  name = "${yandex_container_registry.container_registry.id}/tasker"
}
