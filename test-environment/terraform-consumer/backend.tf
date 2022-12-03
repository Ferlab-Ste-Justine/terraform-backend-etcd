terraform {
  backend "http" {
    update_method = "PUT"
    address = "https://127.0.0.1:14443/state?state=%2Ftest%2Fstate"
    lock_method = "PUT"
    lock_address = "https://127.0.0.1:14443/lock?state=%2Ftest%2Fstate"
    unlock_method = "DELETE"
    unlock_address = "https://127.0.0.1:14443/lock?state=%2Ftest%2Fstate"
    username = "test"
    password = "test"
  }
}

/*terraform {
  backend "etcdv3" {
    endpoints   = ["127.0.0.1:32380"]
    lock        = true
    prefix      = "/test/state/"
    cacert_path = "../etcd-server/certs/ca.pem"
    cert_path   = "../etcd-server/certs/root.pem"
    key_path    = "../etcd-server/certs/root.key"
  }
}*/