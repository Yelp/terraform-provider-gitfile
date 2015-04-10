resource "gitfile_checkout" "test" {
    repo = "${path.root}/example.git/.git"
    branch = "master"
    path = "checkout"
}
output "gitfile_checkout_path" {
    value = "${gitfile_checkout.test.path}"
}
resource "gitfile_file" "test" {
    checkout_dir = "${gitfile_checkout.test.path}"
    path = "terraform"
    contents = "Terraform making commits"
}
resource "gitfile_commit" "test" {
    checkout_dir = "${gitfile_checkout.test.path}"
    commit_message = "Created by terraform gitfile_commit"
    handles = ["${gitfile_file.test.id}"]
}
output "gitfile_commit_commit_message" {
    value = "${gitfile_commit.test.commit_message}"
}
output "gitfile_commit_checkout_dir" {
    value = "${gitfile_commit.test.checkout_dir}"
}
output "gitfile_commit_file" {
    value = "${gitfile_commit.test.file}"
}

