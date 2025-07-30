resource "sftpgo_user" "server" {
  for_each = local.servers

  home_dir = "${var.terraform.sftpgo.home_directory_base}/${each.key}"
  password = random_password.sftpgo[each.key].result
  status   = 1
  username = each.key

  filesystem = {
    provider = 0
  }

  permissions = {
    "/" = "*"
  }
}
