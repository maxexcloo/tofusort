region = "us-west-2"

instance_config = {
  type = "t2.micro"
  ami  = "ami-12345"
  
  security_groups = ["default"]
  
  tags = {
    Owner       = "team"
    Environment = "dev"
  }
}

servers = [
  {
    region = "us-west"
    name   = "web1"
  },
  {
    name   = "web2"
    region = "us-east"
  }
]