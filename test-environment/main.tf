resource "local_file" "hello" {
  content         = "hello8"
  file_permission = "0660"
  filename        = pathexpand("${path.module}/output/hello")
}
