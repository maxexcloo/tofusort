locals {
  tailscale_device_map = {
    for device in data.tailscale_devices.default.devices :
    split(".", device.name)[0] => device
    if length(split(".", device.name)) > 0
  }

  tailscale_devices = {
    for k, server in local.servers : k => {
      fqdn_external = server.fqdn_external
      fqdn_internal = server.fqdn_internal
      private_ipv4  = try([for address in local.tailscale_device_map[k].addresses : address if can(cidrhost("${address}/32", 0))][0], null)
      private_ipv6  = try([for address in local.tailscale_device_map[k].addresses : address if can(cidrhost("${address}/128", 0))][0], null)
    }
    if contains(keys(local.tailscale_device_map), k)
  }

  tailscale_tags = [
    for tag in var.tags : "tag:${tag}"
  ]
}
