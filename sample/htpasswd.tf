resource "htpasswd_password" "server" {
  for_each = local.servers

  password = onepassword_item.server[each.key].password
}
