# Test complex tfvars-style configuration
config = {
  # Scalars first
  domain      = "example.com"
  environment = "production"
  region      = "us-east-1"
  
  # Single-line arrays grouped with scalars  
  availability_zones = ["us-east-1a", "us-east-1b"]
  tags              = ["web", "api", "production"]
  
  # Multi-line objects separated with blank lines
  database = {
    instance_class = "db.t3.micro"
    engine         = "postgresql"
    engine_version = "13.7"
  }

  networking = {
    vpc_cidr = "10.0.0.0/16"
    
    subnets = [
      {
        cidr = "10.0.1.0/24"
        az   = "us-east-1a"
        type = "public"
      },
      {
        az   = "us-east-1b" 
        cidr = "10.0.2.0/24"
        type = "private"
      }
    ]
  }

  applications = {
    web = {
      port        = 80
      protocol    = "HTTP"
      healthcheck = "/health"
    }
    
    api = {
      healthcheck = "/api/health"
      port        = 8080
      protocol    = "HTTP"
    }
  }
}