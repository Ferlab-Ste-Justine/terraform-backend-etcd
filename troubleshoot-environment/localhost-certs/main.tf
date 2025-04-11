module "ca" {
  source = "./ca"
  common_name = var.username
}

resource "tls_private_key" "server" {
  algorithm   = "ECDSA"
  ecdsa_curve = "P384"
}

resource "tls_cert_request" "server" {
  private_key_pem = tls_private_key.server.private_key_pem

  subject {
    common_name  = "localhost"
    organization = var.username
  }

  ip_addresses = ["127.0.0.1"]
}

resource "tls_locally_signed_cert" "server" {
  cert_request_pem   = tls_cert_request.server.cert_request_pem
  ca_private_key_pem = module.ca.key
  ca_cert_pem        = module.ca.certificate

  validity_period_hours = 100*365*24
  early_renewal_hours = 365*24

  allowed_uses = [
    "server_auth",
  ]

  is_ca_certificate = false
}

resource "local_file" "ca_cert" {
  content         = module.ca.certificate
  file_permission = "0660"
  filename        = pathexpand("${path.module}/certs/local_ca.crt")
}

resource "local_file" "server_cert" {
  content         = tls_locally_signed_cert.server.cert_pem
  file_permission = "0660"
  filename        = pathexpand("${path.module}/certs/local_server.crt")
}

resource "local_file" "server_key" {
  content         = tls_private_key.server.private_key_pem
  file_permission = "0660"
  filename        = pathexpand("${path.module}/certs/local_server.key")
}