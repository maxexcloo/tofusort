# Test mixed single-line and multi-line entries with proper spacing
resource "aws_instance" "web" {
  # Scalars should be grouped together
  ami                    = "ami-12345"
  availability_zones     = ["us-east-1a"]
  instance_type          = "t3.micro"
  key_name               = "my-key"
  security_groups        = ["sg-123", "sg-456"]

  # Multi-line objects should be separated with blank lines
  vpc_security_group_ids = [
    "sg-123",
    "sg-456",
    "sg-789"
  ]

  metadata_options {
    http_endpoint = "enabled"
    http_tokens   = "required"
  }

  root_block_device {
    encrypted   = true
    volume_size = 20
    volume_type = "gp3"
  }

  tags = {
    Application = "web"
    Environment = "production"
    Name        = "web-server"
  }
}