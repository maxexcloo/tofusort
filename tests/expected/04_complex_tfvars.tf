# Test complex tfvars-style configuration
config = {
  # Scalars first
  availability_zones = ["us-east-1a", "us-east-1b"]
  domain             = "example.com"
  environment        = "production"
  region             = "us-east-1"
  tags               = ["web", "api", "production"]

  # Multi-line objects separated with blank lines
  applications = {
    api = {
      healthcheck = "/api/health"
      port        = 8080
      protocol    = "HTTP"
    }

    web = {
      healthcheck = "/health"
      port        = 80
      protocol    = "HTTP"
    }
  }

  database = {
    engine         = "postgresql"
    engine_version = "13.7"
    instance_class = "db.t3.micro"
  }

  networking = {
    vpc_cidr = "10.0.0.0/16"

    subnets = [
      {
        az   = "us-east-1a"
        cidr = "10.0.1.0/24"
        type = "public"
      },
      {
        az   = "us-east-1b"
        cidr = "10.0.2.0/24"
        type = "private"
      }
    ]
  }
}