resource "local_file" "hello" {
  count = 10
  content         = "hello${count.index}"
  file_permission = "0660"
  filename        = pathexpand("${path.module}/output/hello${count.index}")
}

resource "local_file" "goodbye" {
  count = 10
  content         = "goodbye${count.index}"
  file_permission = "0660"
  filename        = pathexpand("${path.module}/output/goodbye${count.index}")
}