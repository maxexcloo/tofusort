data "b2_account_info" "default" {}

data "cloudflare_account_api_token_permission_groups_list" "default" {
  account_id = var.terraform.cloudflare.account_id
}

data "cloudflare_zero_trust_tunnel_cloudflared_token" "server" {
  for_each = cloudflare_zero_trust_tunnel_cloudflared.server

  account_id = var.terraform.cloudflare.account_id
  tunnel_id  = each.value.id
}

data "github_user" "default" {
  username = var.terraform.github.username
}

data "oci_core_vnic" "vm" {
  for_each = data.oci_core_vnic_attachments.vm

  vnic_id = each.value.vnic_attachments[0].vnic_id
}

data "oci_core_vnic_attachments" "vm" {
  for_each = oci_core_instance.vm

  compartment_id = var.terraform.oci.tenancy_ocid
  instance_id    = each.value.id
}

data "oci_identity_availability_domain" "au" {
  ad_number      = 1
  compartment_id = var.terraform.oci.tenancy_ocid
}

data "onepassword_vault" "default" {
  name = var.terraform.onepassword.vault
}

data "tailscale_devices" "default" {}
