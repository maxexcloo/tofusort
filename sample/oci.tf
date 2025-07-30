resource "oci_core_default_dhcp_options" "au" {
  compartment_id             = var.terraform.oci.tenancy_ocid
  display_name               = "${var.terraform.oci.location}.${var.default.domain_external}"
  manage_default_resource_id = oci_core_vcn.au.default_dhcp_options_id

  options {
    server_type = "VcnLocalPlusInternet"
    type        = "DomainNameServer"
  }

  options {
    search_domain_names = [oci_core_vcn.au.vcn_domain_name]
    type                = "SearchDomain"
  }
}

resource "oci_core_default_route_table" "au" {
  display_name               = "${var.terraform.oci.location}.${var.default.domain_external}"
  manage_default_resource_id = oci_core_vcn.au.default_route_table_id

  route_rules {
    destination       = "0.0.0.0/0"
    destination_type  = "CIDR_BLOCK"
    network_entity_id = oci_core_internet_gateway.au.id
  }

  route_rules {
    destination       = "::/0"
    destination_type  = "CIDR_BLOCK"
    network_entity_id = oci_core_internet_gateway.au.id
  }
}

resource "oci_core_default_security_list" "au" {
  compartment_id             = var.terraform.oci.tenancy_ocid
  display_name               = "${var.terraform.oci.location}.${var.default.domain_external}"
  manage_default_resource_id = oci_core_vcn.au.default_security_list_id

  dynamic "egress_security_rules" {
    for_each = ["::/0", "0.0.0.0/0"]

    content {
      destination = egress_security_rules.value
      protocol    = "all"
      stateless   = false
    }
  }

  dynamic "ingress_security_rules" {
    for_each = ["::/0", "0.0.0.0/0"]

    content {
      protocol  = 1
      source    = ingress_security_rules.value
      stateless = false
    }
  }

  dynamic "ingress_security_rules" {
    for_each = setproduct(["::/0", "0.0.0.0/0"], flatten([for vm in local.vms_oci : vm.config.ingress_ports]))

    content {
      protocol  = 6
      source    = ingress_security_rules.value[0]
      stateless = false

      tcp_options {
        max = ingress_security_rules.value[1]
        min = ingress_security_rules.value[1]
      }
    }
  }

  dynamic "ingress_security_rules" {
    for_each = setproduct(["::/0", "0.0.0.0/0"], flatten([for vm in local.vms_oci : vm.config.ingress_ports]))

    content {
      protocol  = 17
      source    = ingress_security_rules.value[0]
      stateless = false

      udp_options {
        max = ingress_security_rules.value[1]
        min = ingress_security_rules.value[1]
      }
    }
  }
}

resource "oci_core_instance" "vm" {
  for_each = local.vms_oci

  availability_domain = data.oci_identity_availability_domain.au.name
  compartment_id      = var.terraform.oci.tenancy_ocid
  display_name        = each.key
  shape               = each.value.config.shape

  metadata = {
    user_data = each.value.config.enable_cloud_config ? base64encode(local.output_cloud_config[each.key]) : null
  }

  create_vnic_details {
    assign_ipv6ip             = true
    assign_private_dns_record = true
    assign_public_ip          = true
    display_name              = each.key
    hostname_label            = each.value.name
    subnet_id                 = oci_core_subnet.au.id
  }

  shape_config {
    memory_in_gbs = each.value.config.memory
    ocpus         = each.value.config.cpus
  }

  source_details {
    boot_volume_size_in_gbs = each.value.config.boot_disk_size
    source_id               = each.value.config.boot_disk_image_id
    source_type             = "image"
  }

  lifecycle {
    ignore_changes = [
      metadata
    ]
  }
}

resource "oci_core_internet_gateway" "au" {
  compartment_id = var.terraform.oci.tenancy_ocid
  display_name   = "${var.terraform.oci.location}.${var.default.domain_external}"
  vcn_id         = oci_core_vcn.au.id
}

resource "oci_core_subnet" "au" {
  cidr_block     = "10.0.0.0/24"
  compartment_id = var.terraform.oci.tenancy_ocid
  display_name   = "${var.terraform.oci.location}.${var.default.domain_external}"
  dns_label      = var.terraform.oci.location
  ipv6cidr_block = replace(oci_core_vcn.au.ipv6cidr_blocks[0], "/56", "/64")
  vcn_id         = oci_core_vcn.au.id
}

resource "oci_core_vcn" "au" {
  cidr_blocks    = ["10.0.0.0/16"]
  compartment_id = var.terraform.oci.tenancy_ocid
  display_name   = "${var.terraform.oci.location}.${var.default.domain_external}"
  dns_label      = replace(var.default.domain_external, "/\\.[^.]*$/", "")
  is_ipv6enabled = true
}
