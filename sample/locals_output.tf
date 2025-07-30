locals {
  output_b2 = {
    for k, b2_bucket in b2_bucket.server : k => {
      application_key_id = b2_application_key.server[k].application_key_id
      application_key    = b2_application_key.server[k].application_key
      bucket_name        = b2_bucket.bucket_name
      endpoint           = replace(data.b2_account_info.default.s3_api_url, "https://", "")
    }
  }

  output_cloud_config = {
    for k, server in local.servers : k => templatefile(
      "templates/cloud_config/cloud_config.yaml",
      {
        init_commands = local.output_init_commands[k]
        password_hash = htpasswd_password.server[k].sha512
        server        = server
        ssh_keys      = data.github_user.default.ssh_keys
      }
    )
    if server.config.enable_cloud_config
  }

  output_cloudflare_account_tokens = {
    for k, cloudflare_account_token in cloudflare_account_token.server : k => cloudflare_account_token.value
  }

  output_cloudflare_tunnels = {
    for k, cloudflare_zero_trust_tunnel_cloudflared_token in data.cloudflare_zero_trust_tunnel_cloudflared_token.server : k => {
      cname = "${cloudflare_zero_trust_tunnel_cloudflared_token.tunnel_id}.cfargotunnel.com"
      id    = cloudflare_zero_trust_tunnel_cloudflared_token.tunnel_id
      token = cloudflare_zero_trust_tunnel_cloudflared_token.token
    }
  }

  output_init_commands = {
    for k, server in local.servers : k => concat(
      [
        "sysctl --system"
      ],
      contains(server.config.packages, "qemu-guest-agent") ? [
        "systemctl enable --now qemu-guest-agent"
      ] : [],
      contains(server.flags, "docker") ? [
        "curl -fsLS https://get.docker.com | sh",
        "docker network create ${var.default.organisation}",
      ] : [],
      [
        "curl -fsLS https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-$(dpkg --print-architecture).deb -o /tmp/cloudflared.deb && dpkg -i /tmp/cloudflared.deb && rm /tmp/cloudflared.deb",
        "curl -fsLS https://tailscale.com/install.sh | sh",
        "cloudflared service install ${local.output_cloudflare_tunnels[k].token}",
        "tailscale up --advertise-exit-node --authkey ${local.output_tailscale_tailnet_keys[k]} --hostname ${k}"
      ]
    )
  }

  output_resend_api_keys = {
    for k, restapi_object in restapi_object.resend_api_key_server : k => jsondecode(restapi_object.create_response).token
  }

  output_secret_hashes = {
    for k, random_password in random_password.secret_hash : k => random_password.result
  }

  output_sftpgo = {
    for k, sftpgo_user in sftpgo_user.server : k => {
      home_directory = sftpgo_user.home_dir
      password       = sftpgo_user.password
      username       = sftpgo_user.username
      webdav_url     = var.terraform.sftpgo.webdav_url
    }
  }

  output_tailscale_tailnet_keys = {
    for k, tailscale_tailnet_key in tailscale_tailnet_key.server : k => tailscale_tailnet_key.key
  }
}
