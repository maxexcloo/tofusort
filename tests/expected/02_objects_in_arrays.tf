# Test objects inside arrays should be sorted
variable "servers" {
  type = list(object({
    location = string
    name     = string
    services = list(string)
  }))

  default = [
    {
      location = "us-east"
      name     = "server1"
      services = ["web", "api"]
    },
    {
      location = "us-west"
      name     = "server2"
      services = ["database"]
    }
  ]
}

locals {
  dns_records = [
    {
      content  = "1.2.3.4"
      name     = "api"
      priority = 10
      type     = "A"
    },
    {
      content  = "5.6.7.8"
      name     = "web"
      priority = 5
      type     = "A"
    }
  ]

  # Complex nested array with objects
  infrastructure = [
    {
      region = "us-east-1"
      zones  = ["a", "b", "c"]

      vpc = {
        cidr = "10.0.0.0/16"
        id   = "vpc-123"
      }

      instances = [
        {
          az   = "us-east-1a"
          id   = "i-123"
          type = "t3.micro"
        },
        {
          az   = "us-east-1b"
          id   = "i-456"
          type = "t3.small"
        }
      ]
    }
  ]
}