locals {
  vms_oci = merge({
    for vm in var.vms_oci : "${vm.location}-${vm.name}" => merge(
      var.default.vm_config.base,
      {
        location     = "cloud"
        parent_flags = ["cloud"]
        parent_name  = "oci"
      },
      vm,
      {
        fqdn_external = "${vm.name}.${vm.location}.${var.default.domain_external}"
        fqdn_internal = "${vm.name}.${vm.location}.${var.default.domain_internal}"
        title         = try(vm.title, title(vm.name))
        config = merge(
          var.default.server_config,
          var.default.vm_config.oci,
          try(vm.config, {})
        )
        networks = [
          for network in try(vm.networks, [{}]) : network
        ]
        user = merge(
          var.default.user_config,
          try(vm.user, {})
        )
      }
    )
  })

  vms_proxmox = merge([
    for server in local.servers_physical : {
      for vm in var.vms_proxmox : "${server.location}-${server.name}-${vm.name}" => merge(
        var.default.vm_config.base,
        vm,
        {
          fqdn_external = "${vm.name}.${server.name}.${server.location}.${var.default.domain_external}"
          fqdn_internal = "${vm.name}.${server.name}.${server.location}.${var.default.domain_internal}"
          location      = server.location
          name          = "${server.name}-${vm.name}"
          parent_flags  = server.flags
          parent_name   = server.name
          title         = "${server.title} ${try(vm.title, title(vm.name))}"
          config = merge(
            var.default.server_config,
            var.default.vm_config.proxmox,
            try(vm.config, {}),
            {
              packages = concat(["qemu-guest-agent"], try(vm.config.packages, []), var.default.server_config.packages)
            }
          )
          hostpci = [
            for hostpci in try(vm.hostpci, {}) : merge(
              var.default.vm_config.proxmox_hostpci,
              hostpci
            )
          ]
          networks = [
            for network in try(vm.networks, [{}]) : merge(
              var.default.vm_config.proxmox_network,
              {
                public_address = cloudflare_dns_record.router[server.location].name
              },
              network
            )
          ]
          usb = [
            for usb in try(vm.usb, {}) : merge(
              var.default.vm_config.proxmox_usb,
              usb
            )
          ]
          user = merge(
            var.default.user_config,
            try(vm.user, {})
          )
        },
      )
      if vm.parent == server.name
    }
  ]...)

  vms_vms = merge({
    for vm in var.vms : "${vm.location}-${vm.name}" => merge(
      var.default.vm_config.base,
      {
        location     = "cloud"
        parent_flags = ["cloud"]
        parent_name  = "cloud"
      },
      vm,
      {
        fqdn_external = "${vm.name}.${vm.location}.${var.default.domain_external}"
        fqdn_internal = "${vm.name}.${vm.location}.${var.default.domain_internal}"
        title         = try(vm.title, title(vm.name))
        config = merge(
          var.default.server_config,
          try(vm.config, {})
        )
        networks = [
          for network in try(vm.networks, [{}]) : merge(
            var.default.vm_config.network,
            network
          )
        ]
        user = merge(
          var.default.user_config,
          try(vm.user, {})
        )
      }
    )
  })
}
