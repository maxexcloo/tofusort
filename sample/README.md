# Infrastructure

[![License](https://img.shields.io/badge/license-AGPL--3.0-blue.svg)](LICENSE)
[![Status](https://img.shields.io/badge/status-active-success)](https://img.shields.io/badge/status-active-success)
[![OpenTofu](https://img.shields.io/badge/OpenTofu-blue)](https://opentofu.org/)

OpenTofu configuration for personal infrastructure management.

## Quick Start

1. Copy configuration template:
   ```bash
   cp terraform.tfvars.sample terraform.tfvars
   ```

2. Update `terraform.tfvars` with your values

3. Initialize and apply:
   ```bash
   tofu init
   tofu plan
   tofu apply
   ```

## Features

- **DNS**: Cloudflare zones and automated record creation
- **Networking**: Tailscale mesh with device management  
- **Security**: 1Password integration for secret management
- **Storage**: Backblaze B2 buckets for backup
- **VMs**: Oracle Cloud Infrastructure and Proxmox instances

### Security

- All credentials marked sensitive
- Network access via Tailscale zero-trust
- Secrets managed in 1Password
- State stored in Terraform Cloud

## Installation

### Prerequisites

- OpenTofu 1.6+
- Access to cloud providers (Oracle Cloud, Cloudflare)
- 1Password service account
- Tailscale account

### Setup

1. Clone repository
2. Copy `terraform.tfvars.sample` to `terraform.tfvars`
3. Configure provider credentials in `terraform.tfvars`
4. Run initialization commands

## Usage

### Configuration

Infrastructure defined in `terraform.tfvars`:

```hcl
routers = [
  {
    flags    = ["homepage", "unifi"]
    location = "au"
  }
]

servers = [
  {
    flags  = ["docker", "homepage"] 
    name   = "server-name"
    parent = "router-location"
  }
]

vms_oci = [
  {
    config = {
      cpus   = 4
      memory = 8
    }
    location = "au"
    name     = "vm-name"  
  }
]

vms_proxmox = [
  {
    config = {
      cpus   = 2
      memory = 4
    }
    name   = "vm-name"
    parent = "physical-server-name"
  }
]
```

### Workflow

```bash
tofu fmt && tofu validate && tofu plan
tofu apply
```

### Project Structure

```
├── data.tf                  # All data sources
├── locals_*.tf              # Configuration processing
├── outputs.tf               # Output definitions
├── providers.tf             # Provider configurations
├── terraform.tf             # Terraform configuration
├── variables.tf             # Variable definitions
├── *.tf                     # Resource files
└── terraform.tfvars         # Instance values
```

### Troubleshooting

Common issues:

1. **Authentication errors**: Check `terraform.tfvars` credentials
2. **DNS delays**: Cloudflare changes take time to propagate
3. **Resource conflicts**: Check for naming collisions
4. **VM failures**: Verify cloud provider quotas

Run `tofu validate` to check configuration syntax.

## Contributing

1. Follow the coding standards in `CLAUDE.md`
2. Run `tofu fmt` before committing
3. Ensure all changes pass `tofu validate` and `tofu plan`
4. Use the provided commit message format

## License

This project is licensed under the AGPL-3.0 License - see the [LICENSE](LICENSE) file for details.
