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
