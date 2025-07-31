package sorter

import (
	"testing"
)

// TestRealWorldProvidersFile tests sorting a complex providers.tf based on sample_1
func TestRealWorldProvidersFile(t *testing.T) {
	input := `provider "tfe" {
  # Uses TF_TOKEN environment variable or OpenTofu/Terraform Cloud credentials
}

provider "b2" {
  application_key    = var.terraform.b2.application_key
  application_key_id = var.terraform.b2.application_key_id
}

provider "restapi" {
  alias                = "fly"
  id_attribute         = "id"
  uri                  = var.terraform.fly.url
  write_returns_object = true

  headers = {
    Authorization = "Bearer ${var.terraform.fly.api_token}"
    Content-Type  = "application/json"
  }
}

provider "cloudflare" {
  api_key = var.terraform.cloudflare.api_key
  email   = var.default.email
}

provider "restapi" {
  alias                 = "resend"
  create_returns_object = true
  rate_limit            = 1
  uri                   = var.terraform.resend.url

  headers = {
    Authorization = "Bearer ${var.terraform.resend.api_key}"
    Content-Type  = "application/json"
  }
}

provider "restapi" {
  alias                = "portainer"
  id_attribute         = "Id"
  insecure             = true
  rate_limit           = 10
  uri                  = "${var.terraform.portainer.url}/api"
  write_returns_object = true

  headers = {
    Content-Type = "application/json"
    X-API-Key    = var.terraform.portainer.api_key
  }
}

provider "onepassword" {
  service_account_token = var.terraform.onepassword.service_account_token
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
}`

	expected := `provider "b2" {
  application_key    = var.terraform.b2.application_key
  application_key_id = var.terraform.b2.application_key_id
}

provider "cloudflare" {
  api_key = var.terraform.cloudflare.api_key
  email   = var.default.email
}

provider "onepassword" {
  service_account_token = var.terraform.onepassword.service_account_token
}

provider "restapi" {
  alias                = "fly"
  id_attribute         = "id"
  uri                  = var.terraform.fly.url
  write_returns_object = true

  headers = {
    Authorization = "Bearer ${var.terraform.fly.api_token}"
    Content-Type  = "application/json"
  }
}

provider "restapi" {
  alias                 = "resend"
  create_returns_object = true
  rate_limit            = 1
  uri                   = var.terraform.resend.url

  headers = {
    Authorization = "Bearer ${var.terraform.resend.api_key}"
    Content-Type  = "application/json"
  }
}

provider "restapi" {
  alias                = "portainer"
  id_attribute         = "Id"
  insecure             = true
  rate_limit           = 10
  uri                  = "${var.terraform.portainer.url}/api"
  write_returns_object = true

  headers = {
    Content-Type = "application/json"
    X-API-Key    = var.terraform.portainer.api_key
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
provider "tfe" {
  # Uses TF_TOKEN environment variable or OpenTofu/Terraform Cloud credentials
}
`

	testSorting(t, input, expected)
}

// TestComplexDataBlocks tests sorting data blocks with various types and attributes
func TestComplexDataBlocks(t *testing.T) {
	input := `data "tfe_outputs" "infrastructure" {
  organization = "Excloo"
  workspace    = "Infrastructure"
}

data "onepassword_vault" "services" {
  name = var.terraform.onepassword.vault
}

data "cloudflare_zones" "unique_zones" {
  for_each = local.unique_dns_zones

  name = each.value

  account = {
    id = var.terraform.cloudflare.account_id
  }
}

data "b2_account_info" "default" {}

data "http" "portainer_endpoints" {
  insecure = true
  url      = "${var.terraform.portainer.url}/api/endpoints"

  request_headers = {
    Content-Type = "application/json"
    X-API-Key    = var.terraform.portainer.api_key
  }
}`

	expected := `data "b2_account_info" "default" {}

data "cloudflare_zones" "unique_zones" {
  for_each = local.unique_dns_zones

  name = each.value

  account = {
    id = var.terraform.cloudflare.account_id
  }
}

data "http" "portainer_endpoints" {
  insecure = true
  url      = "${var.terraform.portainer.url}/api/endpoints"

  request_headers = {
    Content-Type = "application/json"
    X-API-Key    = var.terraform.portainer.api_key
  }
}

data "onepassword_vault" "services" {
  name = var.terraform.onepassword.vault
}

data "tfe_outputs" "infrastructure" {
  organization = "Excloo"
  workspace    = "Infrastructure"
}
`

	testSorting(t, input, expected)
}

// TestComplexVariableTypes tests sorting variables with complex nested object types
func TestComplexVariableTypes(t *testing.T) {
	input := `variable "terraform" {
  description = "Provider configurations and credentials"
  sensitive   = true

  type = object({
    tailscale = object({
      oauth_client_id     = string
      oauth_client_secret = string
      organization        = string
    })

    b2 = object({
      application_key    = string
      application_key_id = string
    })

    resend = object({
      smtp_host     = string
      smtp_port     = number
      smtp_username = string
      url           = string
      api_key       = string
    })

    onepassword = object({
      service_account_token = string
      vault                 = string
    })

    cloudflare = object({
      account_id = string
      api_key    = string
    })
  })
}

variable "services" {
  description = "Service configurations using convention over configuration"
  type        = any

  validation {
    error_message = "Service keys must follow 'platform-servicename' pattern."

    condition = alltrue([
      for k, v in var.services : can(regex("^[a-z][a-z0-9-]*-[a-z][a-z0-9-]*$", k))
    ])
  }
}`

	expected := `variable "services" {
  description = "Service configurations using convention over configuration"
  type        = any

  validation {
    error_message = "Service keys must follow 'platform-servicename' pattern."

    condition = alltrue([
      for k, v in var.services : can(regex("^[a-z][a-z0-9-]*-[a-z][a-z0-9-]*$", k))
    ])
  }
}
variable "terraform" {
  description = "Provider configurations and credentials"
  sensitive   = true

  type = object({
    b2 = object({
      application_key    = string
      application_key_id = string
    })

    cloudflare = object({
      account_id = string
      api_key    = string
    })

    onepassword = object({
      service_account_token = string
      vault                 = string
    })

    resend = object({
      api_key       = string
      smtp_host     = string
      smtp_port     = number
      smtp_username = string
      url           = string
    })

    tailscale = object({
      oauth_client_id     = string
      oauth_client_secret = string
      organization        = string
    })
  })
}
`

	testSorting(t, input, expected)
}

// TestMixedBlockTypes tests sorting files with multiple block types
func TestMixedBlockTypes(t *testing.T) {
	input := `provider "b2" {
  application_key    = var.terraform.b2.application_key
  application_key_id = var.terraform.b2.application_key_id
}

variable "tags" {
  default     = {}
  description = "Common tags for resources"
  type        = map(string)
}

data "b2_account_info" "default" {}

terraform {
  required_providers {
    b2 = {
      source  = "Backblaze/b2"
      version = "~> 0.8"
    }
  }
}

locals {
  environment = "production"
}

resource "b2_bucket" "example" {
  bucket_name = "example-bucket"
  bucket_type = "allPrivate"
}

output "bucket_id" {
  value = b2_bucket.example.bucket_id
}`

	expected := `terraform {
  required_providers {
    b2 = {
      source  = "Backblaze/b2"
      version = "~> 0.8"
    }
  }
}

provider "b2" {
  application_key    = var.terraform.b2.application_key
  application_key_id = var.terraform.b2.application_key_id
}

variable "tags" {
  default     = {}
  description = "Common tags for resources"
  type        = map(string)
}

locals {
  environment = "production"
}

data "b2_account_info" "default" {}

resource "b2_bucket" "example" {
  bucket_name = "example-bucket"
  bucket_type = "allPrivate"
}

output "bucket_id" {
  value = b2_bucket.example.bucket_id
}
`

	testSorting(t, input, expected)
}

// TestProxmoxProviderForEach tests complex provider with for_each
func TestProxmoxProviderForEach(t *testing.T) {
	input := `provider "proxmox" {
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
}`

	expected := `provider "proxmox" {
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
`

	testSorting(t, input, expected)
}

// TestComplexTfvarsFile tests .tfvars file sorting
func TestComplexTfvarsFile(t *testing.T) {
	input := `tags = ["production", "infrastructure"]

terraform = {
  b2 = {
    application_key    = "production_key"
    application_key_id = "prod_key_id"
  }
  
  cloudflare = {
    account_id = "cf_account_123"
    api_key    = "cf_api_key_456"
  }
}

default = {
  email        = "admin@example.com"
  name         = "Administrator"
  organisation = "example-org"
}`

	expected := `tags = ["production", "infrastructure"]

default = {
  email        = "admin@example.com"
  name         = "Administrator"
  organisation = "example-org"
}

terraform = {
  b2 = {
    application_key    = "production_key"
    application_key_id = "prod_key_id"
  }

  cloudflare = {
    account_id = "cf_account_123"
    api_key    = "cf_api_key_456"
  }
}
`

	testSorting(t, input, expected)
}

// TestNestedLocalsBlocks tests complex locals with nested structures
func TestNestedLocalsBlocks(t *testing.T) {
	input := `locals {
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
}`

	// For locals, we expect alphabetical sorting of the top-level keys
	expected := `locals {
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
}
`

	testSorting(t, input, expected)
}
