resource "aws_instance" "example" {
  ami           = "ami-12345"
  instance_type = "t2.micro"
  
  lifecycle {
    create_before_destroy = true
  }
  
  depends_on = [aws_security_group.example]
  
  count = 3
  
  tags = {
    Name = "test"
  }
}