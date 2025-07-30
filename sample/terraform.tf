terraform {
  required_version = ">= 1.8"

  backend "remote" {
    hostname     = "app.terraform.io"
    organization = "Excloo"

    workspaces {
      name = "Infrastructure"
    }
  }

  required_providers {
    b2 = {
      source  = "backblaze/b2"
      version = "~> 0.10"
    }

    cloudflare = {
      source  = "cloudflare/cloudflare"
      version = "~> 5.0"
    }

    github = {
      source  = "integrations/github"
      version = "~> 6.0"
    }

    htpasswd = {
      source  = "loafoe/htpasswd"
      version = "~> 1.2"
    }

    local = {
      source  = "hashicorp/local"
      version = "~> 2.5"
    }

    oci = {
      source  = "oracle/oci"
      version = "~> 6.0"
    }

    onepassword = {
      source  = "1password/onepassword"
      version = "~> 2.0"
    }

    proxmox = {
      source  = "bpg/proxmox"
      version = "~> 0.67"
    }

    random = {
      source  = "hashicorp/random"
      version = "~> 3.7"
    }

    restapi = {
      source  = "mastercard/restapi"
      version = "~> 2.0"
    }

    sftpgo = {
      source  = "drakkan/sftpgo"
      version = "~> 0.0.14"
    }

    tailscale = {
      source  = "tailscale/tailscale"
      version = "~> 0.20"
    }
  }
}
