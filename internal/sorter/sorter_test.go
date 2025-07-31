package sorter

import (
	"testing"

	"github.com/yourusername/tofusort/internal/parser"
)

func TestSortSimpleProvider(t *testing.T) {
	input := `provider "test" {
  endpoint = "https://example.com"
  alias    = "test"
  for_each = var.test
}`

	expected := `provider "test" {
  for_each = var.test

  alias    = "test"
  endpoint = "https://example.com"
}
`

	testSorting(t, input, expected)
}

func TestSortMultipleProviders(t *testing.T) {
	input := `provider "z" {
  name = "z"
}

provider "a" {
  name = "a"
}`

	expected := `provider "a" {
  name = "a"
}
provider "z" {
  name = "z"
}
`

	testSorting(t, input, expected)
}

func TestSortBlockTypes(t *testing.T) {
	input := `output "test" {
  value = "test"
}

variable "test" {
  type = string
}

terraform {
  required_version = ">= 1.0"
}`

	expected := `terraform {
  required_version = ">= 1.0"
}
variable "test" {
  type = string
}

output "test" {
  value = "test"
}
`

	testSorting(t, input, expected)
}

func TestBlockTypeOrdering(t *testing.T) {
	input := `resource "aws_instance" "example" {
  ami = "ami-12345"
}

variable "region" {
  default = "us-west-2"
}

terraform {
  required_version = ">= 1.0"
}

provider "aws" {
  region = var.region
}

locals {
  name = "test"
}

data "aws_ami" "ubuntu" {
  most_recent = true
}

module "vpc" {
  source = "./modules/vpc"
}

output "instance_id" {
  value = aws_instance.example.id
}`

	expected := `terraform {
  required_version = ">= 1.0"
}

provider "aws" {
  region = var.region
}

variable "region" {
  default = "us-west-2"
}

locals {
  name = "test"
}

data "aws_ami" "ubuntu" {
  most_recent = true
}

resource "aws_instance" "example" {
  ami = "ami-12345"
}

module "vpc" {
  source = "./modules/vpc"
}

output "instance_id" {
  value = aws_instance.example.id
}
`

	testSorting(t, input, expected)
}

func TestMetaArgumentOrdering(t *testing.T) {
	input := `resource "aws_instance" "example" {
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
}`

	expected := `resource "aws_instance" "example" {
  count = 3

  ami           = "ami-12345"
  instance_type = "t2.micro"

  tags = {
    Name = "test"
  }

  depends_on = [aws_security_group.example]

  lifecycle {
    create_before_destroy = true
  }
}
`

	testSorting(t, input, expected)
}

func TestNestedObjectSorting(t *testing.T) {
	input := `variable "config" {
  default = {
    users = {
      charlie = {
        role = "admin"
        age  = 30
      }
      alice = {
        age  = 25
        role = "user"
      }
      bob = {
        role = "moderator"
        age  = 28
      }
    }
    settings = {
      timeout = 30
      enabled = true
      retries = 3
    }
  }
}`

	expected := `variable "config" {
  default = {
    settings = {
      enabled = true
      retries = 3
      timeout = 30
    }

    users = {
      alice = {
        age  = 25
        role = "user"
      }

      bob = {
        age  = 28
        role = "moderator"
      }

      charlie = {
        age  = 30
        role = "admin"
      }
    }
  }
}
`

	testSorting(t, input, expected)
}

func TestArrayObjectSorting(t *testing.T) {
	input := `locals {
  servers = [
    {
      region = "us-west"
      name   = "server1"
      cpu    = 4
    },
    {
      cpu    = 8
      region = "us-east"
      name   = "server2"
    }
  ]
}`

	expected := `locals {
  servers = [
    {
      cpu    = 4
      name   = "server1"
      region = "us-west"
    },
    {
      cpu    = 8
      name   = "server2"
      region = "us-east"
    }
  ]
}
`

	testSorting(t, input, expected)
}

func TestSingleLineArrayGrouping(t *testing.T) {
	input := `resource "aws_instance" "example" {
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
}`

	expected := `resource "aws_instance" "example" {
  ami                = "ami-12345"
  availability_zones = ["us-west-2a"]
  instance_type      = "t2.micro"
  security_groups    = ["default", "web"]

  root_block_device {
    volume_size = 20
    volume_type = "gp3"
  }

  tags = {
    Application = "web"
    Environment = "production"
  }

  vpc_security_group_ids = [
    "sg-123",
    "sg-456"
  ]
}
`

	testSorting(t, input, expected)
}

func TestTypeObjectSorting(t *testing.T) {
	input := `variable "server" {
  type = object({
    region = string
    cpu    = number
    name   = string
    tags   = map(string)
  })
}`

	expected := `variable "server" {
  type = object({
    cpu    = number
    name   = string
    region = string
    tags   = map(string)
  })
}
`

	testSorting(t, input, expected)
}

func TestTfvarsFileSorting(t *testing.T) {
	input := `region = "us-west-2"

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
]`

	expected := `region = "us-west-2"

instance_config = {
  ami             = "ami-12345"
  security_groups = ["default"]
  type            = "t2.micro"

  tags = {
    Environment = "dev"
    Owner       = "team"
  }
}

servers = [
  {
    name   = "web1"
    region = "us-west"
  },
  {
    name   = "web2"
    region = "us-east"
  }
]
`

	testSorting(t, input, expected)
}

func TestComplexExpressionPreservation(t *testing.T) {
	input := `locals {
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
}`

	expected := `locals {
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
`

	testSorting(t, input, expected)
}

func testSorting(t *testing.T, input, expected string) {
	p := parser.New()
	s := New()

	file, err := p.ParseFile([]byte(input))
	if err != nil {
		t.Fatalf("Failed to parse input: %v", err)
	}

	s.SortFile(file)

	result := string(p.FormatFile(file))
	if result != expected {
		t.Errorf("Sorting failed.\nExpected:\n%s\nGot:\n%s", expected, result)
	}
}
