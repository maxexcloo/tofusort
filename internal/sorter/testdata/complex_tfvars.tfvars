tags = ["production", "infrastructure"]

terraform = {
  b2 = {
    application_key    = "production_key"
    application_key_id = "prod_key_id"
  }
  
  cloudflare = {
    account_id = "cf_account_123"
    api_key    = "cf_api_key_456"
  }
}

default = {
  email        = "admin@example.com"
  name         = "Administrator"
  organisation = "example-org"
}