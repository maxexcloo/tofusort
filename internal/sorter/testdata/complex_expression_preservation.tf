locals {
  servers = {
    for server in var.server_list :
    server.name => {
      cpu    = server.cpu
      region = server.region
    }
  }
  
  tags = merge(
    var.common_tags,
    {
      Environment = "prod"
      Application = "web"
    }
  )
}