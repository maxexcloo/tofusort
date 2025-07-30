resource "tailscale_acl" "default" {
  acl = jsonencode({
    acls = [
      {
        action = "accept"
        ports  = ["*:*"]
        users  = ["*"]
      }
    ]
    autoApprovers = {
      exitNode = local.tailscale_tags
      routes = {
        "0.0.0.0/0" = local.tailscale_tags
        "::/0"      = local.tailscale_tags
      }
    }
    tagOwners = {
      for tag in local.tailscale_tags : tag => [var.default.email]
    }
  })
}

resource "tailscale_tailnet_key" "server" {
  for_each = local.servers

  description   = "${each.value.tag}-${each.key}"
  preauthorized = true
  reusable      = true
  tags          = ["tag:${each.value.tag}"]

  depends_on = [
    tailscale_acl.default
  ]
}
