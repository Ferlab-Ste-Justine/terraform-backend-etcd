module "etcd_ca" {
  source = "./ca"
}

resource "tls_private_key" "etcd" {
  algorithm   = "ECDSA"
  ecdsa_curve = "P384"
}

resource "tls_cert_request" "etcd" {
  private_key_pem = tls_private_key.etcd.private_key_pem
  ip_addresses    = [
    "127.0.0.1",
    "127.0.0.2",
    "127.0.0.3"
  ]
  subject {
    common_name  = "localhost"
    organization = "localhost"
  }
}

resource "tls_locally_signed_cert" "etcd" {
  cert_request_pem   = tls_cert_request.etcd.cert_request_pem
  ca_private_key_pem = module.etcd_ca.key
  ca_cert_pem        = module.etcd_ca.certificate

  validity_period_hours = 10000
  early_renewal_hours = 24

  allowed_uses = [
    "client_auth",
    "server_auth",
  ]

  is_ca_certificate = false
}

module "etcd_root_certificate" {
    source = "git::https://github.com/Ferlab-Ste-Justine/terraform-tls-client-certificate.git"
    ca = module.etcd_ca
    username = "root"
}

resource "local_file" "etcd_ca_cert" {
  content = module.etcd_ca.certificate
  filename = "${path.module}/certs/etcd-ca.crt"
  file_permission = "0600"
}

resource "local_file" "etcd_server_cert" {
  content = tls_locally_signed_cert.etcd.cert_pem
  filename = "${path.module}/certs/etcd-server.crt"
  file_permission = "0600"
}

resource "local_file" "server_key" {
  content = tls_private_key.etcd.private_key_pem
  filename = "${path.module}/certs/etcd-server.key"
  file_permission = "0600"
}

resource "local_file" "root_cert" {
  content = module.etcd_root_certificate.certificate
  filename = "${path.module}/certs/etcd-root.crt"
  file_permission = "0600"
}

resource "local_file" "root_key" {
  content = module.etcd_root_certificate.key
  filename = "${path.module}/certs/etcd-root.key"
  file_permission = "0600"
}