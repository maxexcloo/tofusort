# Test objects inside arrays should be sorted
variable "servers" {
  type = list(object({
    name     = string
    location = string
    services = list(string)
  }))
  
  default = [
    {
      services = ["web", "api"]
      name     = "server1"
      location = "us-east"
    },
    {
      location = "us-west"
      services = ["database"]
      name     = "server2"
    }
  ]
}

locals {
  dns_records = [
    {
      type     = "A"
      name     = "api"
      content  = "1.2.3.4"
      priority = 10
    },
    {
      content  = "5.6.7.8"
      type     = "A" 
      name     = "web"
      priority = 5
    }
  ]
  
  # Complex nested array with objects
  infrastructure = [
    {
      region = "us-east-1"
      zones  = ["a", "b", "c"]
      vpc = {
        id   = "vpc-123"
        cidr = "10.0.0.0/16"
      }
      instances = [
        {
          type = "t3.micro"
          id   = "i-123"
          az   = "us-east-1a"
        },
        {
          az   = "us-east-1b"
          type = "t3.small"
          id   = "i-456"
        }
      ]
    }
  ]
}