resource "keycloak_realm" "chainloop" {
  realm = "chainloop"

  enabled                        = true
  login_with_email_allowed       = true
  registration_email_as_username = true
}

resource "keycloak_openid_client" "chainloop-dev" {
  access_type           = "CONFIDENTIAL"
  client_id             = "chainloop-dev"
  client_secret         = "ZXhhbXBsZS1hcHAtc2VjcmV0"
  enabled               = true
  name                  = "Chainloop dev"
  realm_id              = keycloak_realm.chainloop.id
  standard_flow_enabled = true
  valid_redirect_uris   = ["http://0.0.0.0:8000/auth/callback", "http://localhost:8000/auth/callback"]
}

resource "keycloak_user" "john_chainloop_local" {
  realm_id       = keycloak_realm.chainloop.id
  username       = "john@chainloop.local"
  email          = "john@chainloop.local"
  email_verified = true

  initial_password {
    value = "password"
  }
}

resource "keycloak_user" "sarah_chainloop_local" {
  realm_id       = keycloak_realm.chainloop.id
  username       = "sarah@chainloop.local"
  email          = "sarah@chainloop.local"
  email_verified = true

  initial_password {
    value = "password"
  }
}
