resource "random_password" "password" {
  length           = 16
  special          = false
}

resource "local_file" "password" {
  content         = random_password.password.result
  file_permission = "0660"
  filename        = pathexpand("${path.module}/output/password")
}

resource "local_file" "large_file" {
  content         = file("${path.module}/large_file.txt")
  file_permission = "0660"
  filename        = "${path.module}/output/large_file_copy.txt"
}