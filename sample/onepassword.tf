resource "onepassword_item" "server" {
  for_each = local.servers

  category = "login"
  title    = "${each.key} (${each.value.title})"
  url      = each.key
  username = each.value.user.username
  vault    = data.onepassword_vault.default.uuid

  password_recipe {
    length  = 24
    symbols = false
  }

  section {
    label = "B2"

    field {
      label = "B2 Application Key"
      type  = "CONCEALED"
      value = local.output_b2[each.key].application_key
    }

    field {
      label = "B2 Application Key ID"
      type  = "STRING"
      value = local.output_b2[each.key].application_key_id
    }

    field {
      label = "B2 Bucket Name"
      type  = "STRING"
      value = local.output_b2[each.key].bucket_name
    }

    field {
      label = "B2 Endpoint"
      type  = "URL"
      value = local.output_b2[each.key].endpoint
    }
  }

  section {
    label = "Cloudflare"

    field {
      label = "Cloudflare Account Token"
      type  = "CONCEALED"
      value = local.output_cloudflare_account_tokens[each.key]
    }
  }

  section {
    label = "Cloudflare Tunnel"

    field {
      label = "Cloudflare Tunnel CNAME"
      type  = "URL"
      value = local.output_cloudflare_tunnels[each.key].cname
    }

    field {
      label = "Cloudflare Tunnel ID"
      type  = "STRING"
      value = local.output_cloudflare_tunnels[each.key].id
    }

    field {
      label = "Cloudflare Tunnel Token"
      type  = "CONCEALED"
      value = local.output_cloudflare_tunnels[each.key].token
    }
  }

  section {
    label = "Resend"

    field {
      label = "Resend API Key"
      type  = "CONCEALED"
      value = local.output_resend_api_keys[each.key]
    }
  }

  section {
    label = "Secret Hash"

    field {
      label = "Secret Hash"
      type  = "CONCEALED"
      value = local.output_secret_hashes[each.key]
    }
  }

  section {
    label = "SFTPGo"

    field {
      label = "SFTPGo Username"
      type  = "STRING"
      value = local.output_sftpgo[each.key].username
    }

    field {
      label = "SFTPGo Password"
      type  = "CONCEALED"
      value = local.output_sftpgo[each.key].password
    }

    field {
      label = "SFTPGo Home Directory"
      type  = "STRING"
      value = local.output_sftpgo[each.key].home_directory
    }

    field {
      label = "SFTPGo WebDAV URL"
      type  = "URL"
      value = local.output_sftpgo[each.key].webdav_url
    }
  }

  section {
    label = "Tailscale"

    field {
      label = "Tailscale Tailnet Key"
      type  = "CONCEALED"
      value = local.output_tailscale_tailnet_keys[each.key]
    }
  }

  section {
    label = "URLs"

    dynamic "field" {
      for_each = flatten([
        for k, network in each.value.networks : [
          {
            id    = k
            label = "Public IPv4"
            value = try(network.public_ipv4, null)
          },
          {
            id    = k
            label = "Public IPv6"
            value = try(network.public_ipv6, null)
          },
          {
            id    = k
            label = "Public Address"
            value = try(network.public_address, null)
          }
        ]
      ])

      content {
        label = "${field.value.label}${field.value.id > 0 ? " ${field.value.id + 1}" : ""}"
        type  = "URL"
        value = field.value.value
      }
    }

    field {
      label = "External FQDN"
      type  = "URL"
      value = each.value.fqdn_external
    }

    field {
      label = "Internal FQDN"
      type  = "URL"
      value = each.value.fqdn_internal
    }
  }
}
