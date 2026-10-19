resource "aws_instance" "example" {
  instance_type = "t2.micro"
  ami           = "ami-12345"
  
  security_groups = ["default", "web"]
  
  availability_zones = ["us-west-2a"]
  
  tags = {
    Environment = "production"
    Application = "web"
  }
  
  root_block_device {
    volume_size = 20
    volume_type = "gp3"
  }
  
  vpc_security_group_ids = [
    "sg-123",
    "sg-456"
  ]
}