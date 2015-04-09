resource "gitfile_checkout" "test" {
    repo = "${path.root}/example.git/.git"
    branch = "master"
    path = "checkout"
}

resource "gitfile_commit" "test" {
    checkout_dir = "${gitfile_checkout.test.path}"
    file {
       path = "terraform"
       contents = "Terraform making commits"
   }
}

