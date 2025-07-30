provider "b2" {
  application_key    = var.terraform.b2.application_key
  application_key_id = var.terraform.b2.application_key_id
}

provider "cloudflare" {
  api_key = var.terraform.cloudflare.api_key
  email   = var.default.email
}

provider "github" {
  token = var.terraform.github.token
}

provider "oci" {
  fingerprint  = var.terraform.oci.fingerprint
  private_key  = base64decode(var.terraform.oci.private_key)
  region       = var.terraform.oci.region
  tenancy_ocid = var.terraform.oci.tenancy_ocid
  user_ocid    = var.terraform.oci.user_ocid
}

provider "onepassword" {
  service_account_token = var.terraform.onepassword.service_account_token
}

provider "proxmox" {
  for_each = nonsensitive(var.terraform.proxmox)

  alias    = "by_host"
  endpoint = "https://${each.value.host}:${each.value.port}"
  insecure = true
  password = each.value.password
  username = "${each.value.username}@pam"

  ssh {
    agent    = true
    username = each.value.username

    node {
      address = each.value.host
      name    = each.key
    }
  }
}

provider "restapi" {
  alias                 = "resend"
  create_returns_object = true
  rate_limit            = 1
  uri                   = var.terraform.resend.url

  headers = {
    "Authorization" = "Bearer ${var.terraform.resend.api_key}",
    "Content-Type"  = "application/json"
  }
}

provider "sftpgo" {
  host     = var.terraform.sftpgo.host
  password = var.terraform.sftpgo.password
  username = var.terraform.sftpgo.username
}

provider "tailscale" {
  oauth_client_id     = var.terraform.tailscale.oauth_client_id
  oauth_client_secret = var.terraform.tailscale.oauth_client_secret
  tailnet             = var.terraform.tailscale.organization
}
