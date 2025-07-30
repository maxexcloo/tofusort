locals {
  dns = merge([
    for zone, records in var.dns : {
      for i, record in records : "${record.name == "@" ? "" : "${record.name}."}${zone}-${lower(record.type)}-${i}" => merge(
        {
          priority = null
          wildcard = false
          zone     = zone
        },
        record
      )
    }
  ]...)

  dns_cloudflare_record_wildcard = merge(
    {
      for k, cloudflare_dns_record in cloudflare_dns_record.dns : k => cloudflare_dns_record
      if local.dns[k].wildcard
    },
    {
      for k, cloudflare_dns_record in cloudflare_dns_record.internal_ipv4 : "${k}-internal" => cloudflare_dns_record
    },
    cloudflare_dns_record.noncloud,
    cloudflare_dns_record.vm_ipv4,
    cloudflare_dns_record.vm_oci_ipv4
  )
}
