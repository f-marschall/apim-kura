terraform {
  backend "azurerm" {
    resource_group_name  = "apim-kura"
    storage_account_name = "apimkura"
    container_name       = "devops"
    key                  = "apim.tfstate"
  }
}
