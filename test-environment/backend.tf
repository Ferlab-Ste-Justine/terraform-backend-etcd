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