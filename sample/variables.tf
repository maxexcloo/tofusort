variable "default" {
  description = "Default configuration values including domains, email, and infrastructure defaults"

  default = {
    domain_external = "excloo.net"
    domain_internal = "excloo.org"
    domain_root     = "excloo.com"
    email           = "max@excloo.com"
    name            = "Max Schaefer"
    organisation    = "excloo"
    server_config = {
      enable_cloud_config      = false
      enable_ssh_password_auth = false
      locale                   = "en_US"
      packages                 = ["curl", "sudo"]
      ssh_port                 = 22
      timezone                 = "UTC"
    }
    user_config = {
      docker_path = "/var/lib/docker"
      fullname    = ""
      groups      = ["docker", "sudo"]
      paths       = []
      shell       = "/bin/bash"
      username    = "root"
    }
    vm_config = {
      base = {
        flags    = []
        services = []
        tag      = "vm"
      }
      network = {
        public_ipv4 = ""
        public_ipv6 = ""
      }
      oci = {
        boot_disk_image_id = ""
        boot_disk_size     = 128
        cpus               = 4
        ingress_ports      = [22, 80, 443]
        memory             = 8
        shape              = "VM.Standard.A1.Flex"
      }
      proxmox = {
        boot_disk_image_compression_algorithm = null
        boot_disk_image_url                   = ""
        boot_disk_size                        = 128
        cpus                                  = 4
        enable_serial                         = false
        memory                                = 8
        operating_system                      = "l26"
      }
      proxmox_hostpci = {
        pcie = true
        xvga = false
      }
      proxmox_network = {
        firewall = true
        vlan_id  = null
      }
      proxmox_usb = {
        usb3 = true
      }
    }
  }

  type = object({
    domain_external = string
    domain_internal = string
    domain_root     = string
    email           = string
    name            = string
    organisation    = string
    server_config   = any
    user_config     = any
    vm_config       = any
  })
}

variable "devices" {
  description = "Device configurations for network infrastructure"
  type        = any

  validation {
    error_message = "Device configurations cannot be null or empty."

    condition = alltrue([
      for k, v in var.devices : v != null && v != {}
    ])
  }

  validation {
    error_message = "Device names must start with a lowercase letter and contain only lowercase letters, numbers, and hyphens."

    condition = alltrue([
      for device in var.devices : can(regex("^[a-z][a-z0-9-]*$", device.name))
    ])
  }
}

variable "dns" {
  description = "DNS record configurations for all domains and zones"
  type        = any

  validation {
    error_message = "DNS configurations cannot be null or empty."

    condition = alltrue([
      for k, v in var.dns : v != null && v != {}
    ])
  }
}

variable "routers" {
  description = "Router configurations for network infrastructure"
  type        = any

  validation {
    error_message = "Router configurations cannot be null or empty."

    condition = alltrue([
      for k, v in var.routers : v != null && v != {}
    ])
  }
}

variable "servers" {
  description = "Server configurations for infrastructure deployment"
  type        = any

  validation {
    error_message = "Server configurations cannot be null or empty."

    condition = alltrue([
      for k, v in var.servers : v != null && v != {}
    ])
  }

  validation {
    error_message = "Server names must contain only lowercase letters, numbers, and hyphens, starting with a letter."

    condition = alltrue([
      for k, v in var.servers : can(regex("^[a-z][a-z0-9-]*$", try(v.name, k)))
    ])
  }
}

variable "tags" {
  default     = []
  description = "Common tags to apply to all infrastructure resources"
  type        = list(string)

  validation {
    error_message = "Tags must start with a letter and contain only alphanumeric characters, underscores, and hyphens."

    condition = alltrue([
      for tag in var.tags : can(regex("^[a-zA-Z][a-zA-Z0-9_-]*$", tag))
    ])
  }
}

variable "terraform" {
  description = "Terraform provider configurations and API credentials"
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
    github = object({
      repository = string
      token      = string
      username   = string
    })
    oci = object({
      fingerprint  = string
      location     = string
      private_key  = string
      region       = string
      tenancy_ocid = string
      user_ocid    = string
    })
    onepassword = object({
      service_account_token = string
      vault                 = string
    })
    proxmox = object({
      pie = object({
        host     = string
        password = string
        port     = number
        username = string
      })
    })
    resend = object({
      api_key = string
      url     = string
    })
    sftpgo = object({
      home_directory_base = string
      host                = string
      password            = string
      username            = string
      webdav_url          = string
    })
    tailscale = object({
      oauth_client_id     = string
      oauth_client_secret = string
      organization        = string
    })
  })
}

variable "vms" {
  description = "Virtual machine configurations for all platforms"
  type        = any

  validation {
    error_message = "VM configurations cannot be null or empty."

    condition = alltrue([
      for k, v in var.vms : v != null && v != {}
    ])
  }
}

variable "vms_oci" {
  description = "Oracle Cloud Infrastructure virtual machine configurations"
  type        = any

  validation {
    error_message = "OCI VM configurations cannot be null or empty."

    condition = alltrue([
      for k, v in var.vms_oci : v != null && v != {}
    ])
  }
}

variable "vms_proxmox" {
  description = "Proxmox virtual machine configurations"
  type        = any

  validation {
    error_message = "Proxmox VM configurations cannot be null or empty."

    condition = alltrue([
      for k, v in var.vms_proxmox : v != null && v != {}
    ])
  }
}
