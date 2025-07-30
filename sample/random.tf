resource "random_password" "b2" {
  for_each = local.servers

  length  = 6
  special = false
  upper   = false
}

resource "random_password" "secret_hash" {
  for_each = local.servers

  length  = 24
  special = false
}

resource "random_password" "sftpgo" {
  for_each = local.servers

  length  = 24
  special = false
}
