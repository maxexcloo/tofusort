locals {
  servers = merge(
    local.servers_routers,
    local.servers_physical,
    local.vms_vms,
    local.vms_oci,
    local.vms_proxmox
  )

  servers_devices = {
    for device in var.devices : device.name => merge(
      {
        port     = 22
        username = "root"
      },
      device
    )
  }

  servers_onprem = merge(
    local.servers_physical,
    local.vms_proxmox
  )

  servers_physical = merge([
    for router in local.servers_routers : {
      for server in var.servers : "${router.location}-${server.name}" => merge(
        {
          flags    = []
          services = []
          tag      = "server"
        },
        server,
        {
          fqdn_external = "${server.name}.${router.location}.${var.default.domain_external}"
          fqdn_internal = "${server.name}.${router.location}.${var.default.domain_internal}"
          location      = router.location
          parent_flags  = router.flags
          parent_name   = router.name
          title         = try(server.title, title(server.name))
          config = merge(
            var.default.server_config,
            try(server.config, {})
          )
          networks = [
            for network in try(server.networks, [{}]) : merge(
              {
                public_address = cloudflare_dns_record.router[router.location].name
              },
              network
            )
          ]
          user = merge(
            var.default.user_config,
            try(server.user, {})
          )
        },
      )
      if server.parent == router.name
    }
  ]...)

  servers_routers = {
    for router in var.routers : router.location => merge(
      {
        flags        = []
        parent_flags = []
        parent_name  = ""
        services     = []
        tag          = "router"
      },
      router,
      {
        fqdn_external = "${router.location}.${var.default.domain_external}"
        fqdn_internal = "${router.location}.${var.default.domain_internal}"
        name          = router.location
        title         = try(router.title, upper(router.name), upper(router.location))
        config = merge(
          var.default.server_config,
          try(router.config, {})
        )
        networks = [
          for network in try(router.networks, [{}]) : merge(
            {
              public_address = ""
            },
            network
          )
        ]
        user = merge(
          var.default.user_config,
          try(router.user, {})
        )
      }
    )
  }
}
