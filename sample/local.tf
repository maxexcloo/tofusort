resource "local_file" "ssh_config" {
  filename = "../../.ssh/config"

  content = templatefile(
    "templates/ssh/config",
    {
      devices = local.servers_devices
      servers = local.servers
    }
  )
}
