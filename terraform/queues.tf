resource "yandex_message_queue" "connector-events" {
  name                       = "connector-events.fifo"
  visibility_timeout_seconds = 30
  receive_wait_time_seconds  = 20
  fifo_queue                 = true
  access_key                 = local.aws_secrets_map["access_key_id"]
  secret_key                 = local.aws_secrets_map["secret_key"]
}

resource "yandex_message_queue" "tasker-eval-events" {
  name                       = "tasker-eval-events.fifo"
  visibility_timeout_seconds = 30
  receive_wait_time_seconds  = 20
  fifo_queue                 = true
  access_key                 = local.aws_secrets_map["access_key_id"]
  secret_key                 = local.aws_secrets_map["secret_key"]
}

resource "yandex_message_queue" "notificator-commands" {
  name                       = "notificator-commands"
  visibility_timeout_seconds = 10
  receive_wait_time_seconds  = 20
  access_key                 = local.aws_secrets_map["access_key_id"]
  secret_key                 = local.aws_secrets_map["secret_key"]
}

data "yandex_lockbox_secret_version" "aws_secret" {
  secret_id  = "e6q869h32umj7dap12qa"
  version_id = "e6q1l84sfb3f05qqu03e"
}

locals {
  aws_secrets_map = {
    for e in data.yandex_lockbox_secret_version.aws_secret.entries :
    e.key => e.text_value
  }
}
