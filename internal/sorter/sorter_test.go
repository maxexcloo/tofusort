package sorter

import (
	"strings"
	"testing"

	"github.com/maxexcloo/tofusort/internal/parser"
)

func TestSortPreservesCommentedBodiesAndExpressions(t *testing.T) {
	input := `# File header

locals {
  # Keep this with z.
  z = 1 # Inline note

  a = {
    # Object note
    z = 1
    a = 2
  }

  # Footer inside locals
}

# File footer
`
	p := parser.New()
	file, err := p.ParseFile([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	if err := New().SortFile(file); err != nil {
		t.Fatal(err)
	}
	output := string(p.FormatFile(file))
	for _, comment := range []string{"# File header", "# Keep this with z.", "# Inline note", "# Object note", "# Footer inside locals", "# File footer"} {
		if strings.Count(output, comment) != 1 {
			t.Errorf("comment %q was lost or duplicated:\n%s", comment, output)
		}
	}
	if strings.Index(output, "# File header") > strings.Index(output, "locals") ||
		strings.Index(output, "# File footer") < strings.LastIndex(output, "}") ||
		!strings.Contains(output, "z = 1 # Inline note") ||
		!strings.Contains(output, "# Object note\n    z = 1") {
		t.Errorf("comment attachment changed:\n%s", output)
	}
}

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

  tags = {
    Application = "web"
    Environment = "production"
  }

  vpc_security_group_ids = [
    "sg-123",
    "sg-456"
  ]

  root_block_device {
    volume_size = 20
    volume_type = "gp3"
  }
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
      Application = "web"
      Environment = "prod"
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

	if err := s.SortFile(file); err != nil {
		t.Fatal(err)
	}

	result := string(p.FormatFile(file))
	if withoutBlankLines(result) != withoutBlankLines(expected) {
		t.Errorf("Sorting failed.\nExpected:\n%s\nGot:\n%s", expected, result)
	}
}

func TestSortPreservesComplexNestedExpressions(t *testing.T) {
	inputs := []string{
		`locals {
 policies = [{ resources = jsonencode({ for zone in var.zones : "${zone}" => "*" }) }]
}`,
		`locals {
 schema = { "Compute.Firmware" = jsonencode({ defaultValue = "UEFI_64" }) }
}`,
		`locals {
 names = { (var.key) = "last", "a.key" = "first" }
}`,
	}
	for _, input := range inputs {
		t.Run(input, func(t *testing.T) {
			p := parser.New()
			file, err := p.ParseFile([]byte(input))
			if err != nil {
				t.Fatal(err)
			}
			original := string(p.FormatFile(file))
			if err := New().SortFile(file); err != nil {
				t.Fatal(err)
			}
			result := p.FormatFile(file)
			again, err := p.ParseFile(result)
			if err != nil {
				t.Fatalf("sort produced invalid HCL: %v\n%s", err, result)
			}
			if string(result) != original {
				t.Fatalf("unsupported expression changed:\n%s", result)
			}
			if err := New().SortFile(again); err != nil {
				t.Fatal(err)
			}
			if string(p.FormatFile(again)) != string(result) {
				t.Fatal("sort is not idempotent")
			}
		})
	}
}

func TestSortSingleLineBlock(t *testing.T) {
	p := parser.New()
	file, err := p.ParseFile([]byte(`locals { name = "value" }`))
	if err != nil {
		t.Fatal(err)
	}
	if err := New().SortFile(file); err != nil {
		t.Fatal(err)
	}
	output := p.FormatFile(file)
	if _, err := p.ParseFile(output); err != nil {
		t.Fatalf("invalid output: %v\n%s", err, output)
	}
}

func withoutBlankLines(src string) string {
	var lines []string
	for _, line := range strings.Split(src, "\n") {
		if strings.TrimSpace(line) != "" {
			lines = append(lines, line)
		}
	}
	return strings.Join(lines, "\n")
}
