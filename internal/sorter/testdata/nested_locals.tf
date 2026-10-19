locals {
  services_merged = {
    for k, service in local.services_all : k => merge(
      service,
      {
        # DNS configuration
        dns_content             = local.services_dns_config[k].content
        enable_cloudflare_proxy = contains(try(local.services_computations[k].server_config.flags, []), "cloudflare_proxy")
        enable_dns              = local.services_computations[k].has_dns
        
        # FQDN and URL
        fqdn = local.services_fqdn_config[k].fqdn
        url = local.services_computations[k].has_dns || local.services_computations[k].has_server ? (
          "${try(service.ssl, true) ? "https://" : "http://"}${local.services_fqdn_config[k].base_hostname}"
        ) : null
      }
    )
  }

  output_servers = nonsensitive(jsondecode(data.tfe_outputs.infrastructure.values.servers))
  
  config_outputs = merge(
    {
      for k, service in local.services_merged : k => {
        "/app/config.yaml" = templatefile(
          "templates/gatus/config.yaml",
          {
            default   = var.default
            gatus     = service
            terraform = var.terraform
          }
        )
      }
      if try(service.service, "") == "gatus"
    }
  )
}