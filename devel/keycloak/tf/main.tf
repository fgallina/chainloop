provider "keycloak" {
  client_id = "admin-cli"
  url       = "http://keycloak:8080"
  username  = "admin"
  password  = "admin"
}
