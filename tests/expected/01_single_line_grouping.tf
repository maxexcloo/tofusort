# Test single-line arrays and objects grouping with scalars
variable "test" {
  default     = "test"
  description = "Test variable"
  sensitive   = false
  type        = string

  validation {
    condition     = length(var.test) > 0
    error_message = "Test cannot be empty"
  }
}

locals {
  # This should group single-line arrays with scalars
  config = {
    domain       = "example.com"
    enabled      = true
    environments = ["dev", "prod"]
    port         = 8080
    tags         = ["web", "api"]

    # Multi-line objects should be separated with blank lines
    database = {
      host     = "localhost"
      port     = 5432
      username = "admin"
    }

    services = {
      api = {
        port = 8000
        ssl  = false
      }

      web = {
        port = 3000
        ssl  = true
      }
    }
  }
}