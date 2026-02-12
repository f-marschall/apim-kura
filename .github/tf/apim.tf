terraform {
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "~> 3.0"
    }
  }
}

provider "azurerm" {
  features {}
}

variable "environment" {
  description = "Environment name, typically the branch name"
  type        = string
}

variable "product_count" {
  description = "Number of products to create"
  type        = number
  default     = 3
}

variable "subscription_key_count" {
  description = "Number of subscription keys to create for each product"
  type        = number
  default     = 3
}

# Data source to reference existing resource group
data "azurerm_resource_group" "main" {
  name = "apim-kura"
}

resource "azurerm_api_management" "main" {
  # name                = "apim-${var.environment}-${random_string.unique.result}"
  name                = "gh-apim-kura-${var.environment}"
  location            = data.azurerm_resource_group.main.location
  resource_group_name = data.azurerm_resource_group.main.name
  publisher_name      = "APIM Kura"
  publisher_email     = "apimkura@fmarschall.com"
  sku_name            = "Consumption_0"

  tags = {
    environment = var.environment
  }
}

# Random string for unique naming
resource "random_string" "unique" {
  length  = 4
  special = false
  upper   = false
}

# Sample API
resource "azurerm_api_management_api" "sample_api" {
  name                = "sample-api"
  resource_group_name = data.azurerm_resource_group.main.name
  api_management_name = azurerm_api_management.main.name
  revision            = "1"
  display_name        = "Sample API"
  path                = "sample"
  protocols           = ["https"]
  service_url         = "https://httpbin.org"
}

# Sample API Operation
resource "azurerm_api_management_api_operation" "sample_get" {
  operation_id        = "get-sample"
  api_name            = azurerm_api_management_api.sample_api.name
  api_management_name = azurerm_api_management.main.name
  resource_group_name = data.azurerm_resource_group.main.name
  display_name        = "Get Sample"
  method              = "GET"
  url_template        = "/get"

  response {
    status_code = 200
  }
}

# Products
resource "azurerm_api_management_product" "sample_product" {
  for_each = {
    for i in range(var.product_count) : "product-${i + 1}" => i + 1
  }

  product_id            = "sample-product-${each.value}"
  api_management_name   = azurerm_api_management.main.name
  resource_group_name   = data.azurerm_resource_group.main.name
  display_name          = "Sample Product ${each.value}"
  subscription_required = true
  approval_required     = false
  published             = true
}

# Link API to Products
resource "azurerm_api_management_product_api" "sample_product_api" {
  for_each = azurerm_api_management_product.sample_product

  api_name            = azurerm_api_management_api.sample_api.name
  product_id          = each.value.product_id
  api_management_name = azurerm_api_management.main.name
  resource_group_name = data.azurerm_resource_group.main.name
}

# Global Subscription Keys
resource "azurerm_api_management_subscription" "global_subscription" {
  for_each = {
    for i in range(var.subscription_key_count) : "global-${i}" => i
  }

  api_management_name = azurerm_api_management.main.name
  resource_group_name = data.azurerm_resource_group.main.name
  # user_id             = azurerm_api_management_user.sample_user.id
  display_name        = "Global Subscription ${each.value + 1}"
  state               = "active"

  # depends_on = [azurerm_api_management_user.sample_user]
}

# Product Subscription Keys
resource "azurerm_api_management_subscription" "product_subscription" {
  for_each = {
    for combo in flatten([
      for product_key, product in azurerm_api_management_product.sample_product : [
        for i in range(var.subscription_key_count) : "${product_key}-key-${i + 1}"
      ]
    ]) : combo => combo
  }

  api_management_name = azurerm_api_management.main.name
  resource_group_name = data.azurerm_resource_group.main.name
  # user_id             = azurerm_api_management_user.sample_user.id
  product_id          = azurerm_api_management_product.sample_product[split("-key-", each.key)[0]].id
  display_name        = "Product Subscription ${each.key}"
  state               = "active"

  depends_on = [
    azurerm_api_management_subscription.global_subscription,
    azurerm_api_management_product_api.sample_product_api
  ]
}

# API Subscription Keys
resource "azurerm_api_management_subscription" "api_subscription" {
  for_each = {
    for combo in flatten([
      for product_key, product in azurerm_api_management_product.sample_product : [
        for i in range(var.subscription_key_count) : "${product_key}-api-${i + 1}"
      ]
    ]) : combo => combo
  }

  api_management_name = azurerm_api_management.main.name
  resource_group_name = data.azurerm_resource_group.main.name
  # user_id             = azurerm_api_management_user.sample_user.id
  display_name        = "API Subscription ${each.key}"
  state               = "active"

  depends_on = [
    azurerm_api_management_subscription.product_subscription,
    azurerm_api_management_product_api.sample_product_api
  ]
}

# (Because of the pricing tier consuption, we cannot create users and link them to subscriptions, so we create "orphan" subscriptions without users. In a real scenario, you would create users and link them to these subscriptions.)
# Sample User for Subscriptions
# resource "azurerm_api_management_user" "sample_user" {
#   api_management_name = azurerm_api_management.main.name
#   resource_group_name = data.azurerm_resource_group.main.name
#   user_id             = "sample-user"
#   first_name          = "Sample"
#   last_name           = "User"
#   email               = "sample@example.com"
#   state               = "active"
# }

# Outputs
output "api_management_id" {
  value       = azurerm_api_management.main.id
  description = "The ID of the API Management Service"
}

output "api_management_name" {
  value       = azurerm_api_management.main.name
  description = "The name of the API Management Service"
}

output "gateway_url" {
  value       = azurerm_api_management.main.gateway_url
  description = "The gateway URL of the API Management Service"
}

output "global_subscription_keys" {
  value = {
    for key, sub in azurerm_api_management_subscription.global_subscription : key => sub.primary_key
  }
  sensitive   = true
  description = "Primary keys for all global subscriptions"
}

output "product_subscription_keys" {
  value = {
    for key, sub in azurerm_api_management_subscription.product_subscription : key => sub.primary_key
  }
  sensitive   = true
  description = "Primary keys for all product subscriptions"
}

output "api_subscription_keys" {
  value = {
    for key, sub in azurerm_api_management_subscription.api_subscription : key => sub.primary_key
  }
  sensitive   = true
  description = "Primary keys for all API subscriptions"
}
