# Test single-line arrays and objects grouping with scalars
variable "test" {
  description = "Test variable"
  type        = string
  default     = "test"
  sensitive   = false
  
  validation {
    condition     = length(var.test) > 0
    error_message = "Test cannot be empty"
  }
}

locals {
  # This should group single-line arrays with scalars
  config = {
    domain      = "example.com"
    enabled     = true
    port        = 8080
    
    # These single-line arrays should be grouped with scalars above, not separated
    tags        = ["web", "api"]
    environments = ["dev", "prod"]
    
    # Multi-line objects should be separated with blank lines
    database = {
      host     = "localhost"
      port     = 5432
      username = "admin"
    }
    
    services = {
      web = {
        port = 3000
        ssl  = true
      }
      api = {
        port = 8000
        ssl  = false
      }
    }
  }
}