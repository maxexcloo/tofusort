#!/bin/bash

# Edge case tests for tofusort
set -e

echo "Running edge case tests..."
echo "=========================="

# Build the tool
echo "Building tofusort..."
mise exec -- go build -o tofusort ./cmd/tofusort

echo ""
echo "Test 1: Nested object sorting within default blocks"
echo "---"
cat > /tmp/test1.tf << 'EOF'
variable "test" {
  default = {
    users = {
      charlie = { role = "admin", age = 30 }
      alice = { age = 25, role = "user" }
      bob = { role = "moderator", age = 28 }
    }
    settings = {
      timeout = 30
      enabled = true
      retries = 3
    }
  }
}
EOF

echo "Before:"
cat /tmp/test1.tf | head -20

mise exec -- ./tofusort sort /tmp/test1.tf > /dev/null 2>&1

echo -e "\nAfter:"
cat /tmp/test1.tf | head -20

echo ""
echo "Test 2: Single-line arrays should group with scalars"
echo "---"
cat > /tmp/test2.tf << 'EOF'
resource "test" "example" {
  name = "test"
  enabled = true
  
  tags = ["web", "api"]
  
  ports = [80, 443]
  
  config = {
    timeout = 30
  }
}
EOF

echo "Before:"
cat /tmp/test2.tf

mise exec -- ./tofusort sort /tmp/test2.tf > /dev/null 2>&1

echo -e "\nAfter:"
cat /tmp/test2.tf

echo ""
echo "Test 3: Complex expression preservation"
echo "---"
cat > /tmp/test3.tf << 'EOF'
locals {
  servers = {
    for s in var.servers : s.name => {
      region = s.region
      cpu = s.cpu
      memory = s.memory
    } if s.enabled
  }
}
EOF

echo "Before:"
cat /tmp/test3.tf

mise exec -- ./tofusort sort /tmp/test3.tf > /dev/null 2>&1

echo -e "\nAfter:"
cat /tmp/test3.tf

echo ""
echo "Test 4: Type object() sorting"
echo "---"
cat > /tmp/test4.tf << 'EOF'
variable "server" {
  type = object({
    zone = string
    region = string
    cpu = number
    name = string
    memory = number
  })
}
EOF

echo "Before:"
cat /tmp/test4.tf

mise exec -- ./tofusort sort /tmp/test4.tf > /dev/null 2>&1

echo -e "\nAfter:"
cat /tmp/test4.tf

echo ""
echo "Test 5: Multiple validation blocks"
echo "---"
cat > /tmp/test5.tf << 'EOF'
variable "test" {
  type = string
  
  validation {
    condition = length(var.test) > 0
    error_message = "Cannot be empty"
  }
  
  validation {
    error_message = "Must start with letter"
    condition = can(regex("^[a-z]", var.test))
  }
  
  validation {
    condition = length(var.test) < 20
    error_message = "Too long"
  }
}
EOF

echo "Before:"
cat /tmp/test5.tf

mise exec -- ./tofusort sort /tmp/test5.tf > /dev/null 2>&1

echo -e "\nAfter:"
cat /tmp/test5.tf

# Cleanup
rm -f /tmp/test*.tf

echo ""
echo "=========================="
echo "Edge case tests completed!"