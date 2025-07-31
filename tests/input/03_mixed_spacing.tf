# Test mixed single-line and multi-line entries with proper spacing
resource "aws_instance" "web" {
  # Scalars should be grouped together
  instance_type = "t3.micro"
  ami           = "ami-12345"
  key_name      = "my-key"
  
  # Single-line arrays should be grouped with scalars
  security_groups = ["sg-123", "sg-456"]
  availability_zones = ["us-east-1a"]
  
  # Multi-line objects should be separated with blank lines
  vpc_security_group_ids = [
    "sg-123",
    "sg-456",
    "sg-789"
  ]

  root_block_device {
    volume_type = "gp3"
    volume_size = 20
    encrypted   = true
  }

  metadata_options {
    http_endpoint = "enabled"
    http_tokens   = "required"
  }

  tags = {
    Environment = "production"
    Application = "web"
    Name        = "web-server"
  }
}