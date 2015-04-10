# terraform-provider-gitfile

## Synopsis

A [Terraform](http://terraform.io) plugin to manage files in git repositories.

This allows you to export terraform managed state into other systems which are controlled
by git repositories - for example commit server IPs to DNS config repositories,
or write out hiera data into your puppet configuration.

## Example:


    resource "gitfile_checkout" "example" {
        repo = "git@yourcompany.com:example"
        branch = "master"
        path = "${path.root}/../other_git_checkouts/example"
    }

    resource "aws_instance" "importantbox" {
        ....
        count = 5
    }

    resource "gitfile_file" "importantbox" {
        count = 5
        checkout_dir = "${gitfile_checkout.example.path}"
        path = "directory_of_ips/${element(aws_instance.importantbox.*.private_ip, count.index)}"
        contents = "this is a super important box"
    }

    resource "gitfile_commit" "importantboxes" {
        checkout_dir = "${gitfile_checkout.example.path}"
        commit_message = "Added example IP files for some important boxes"
        handles = ["${gitfile_file.importantbox.*.id}"]
    }

## Resources

### gitfile_checkout

Checks out a git repository onto your local filesystem from within a terraform provider.

This is mostly used to ensure that a checkout is present, before using the _gitfile_commit_
resource to commit some Terraform generated data.

Inputs:

  - repo - The git path to the repository, this can be anything you can feed to 'git clone'
  - branch - The branch to checkout, defaults to 'master'
  - path - The file path on filesystem for where to put the checkout

Outputs:

  - path - The file path on filesystem where the repository has been checked out

### gitfile_file

Creates a file within a git repository with some content from terraform

Inputs:

  - checkout_dir - The path to a git checkout, this can have been made by _gitfile_checkout_ or any other mechanism.
  - count - The number of files to create (so you can create one file per resource for other sets of resources)
  - path - The path within the checkout to create the file at
  - contents - The contents of the file

Outputs:

  - id - The id of the created file. This is usually passed to _gitfile_commit_

### gitfile_symlink

Creates a symlink within a git repository from terraform

Inputs:

  - checkout_dir - The path to a git checkout, this can have been made by _gitfile_checkout_ or any other mechanism.
  - count - The number of symlinks to create (so you can create one symlink per resource for other sets of resources)
  - path - The path within the checkout to create the symlink at
  - target - The place the symlink should point to. Can be an absolute or relative path.

Outputs:

  - - id - The id of the created symlink. This is usually passed to _gitfile_commit_

### gitfile_commit

Makes a git commit of a set of _gitfile_commit_ and _gitfile_file_ resources in a git
repository, and pushes it to origin.

Note that even if the a file with the same contents Terraform creates already exists,
Terraform will create an empty commit with the specified commit message.

Inputs:

  - checkout_dir - The path to a git checkout, this can have been made by _gitfile_checkout_ or any other mechanism.
  - commit_message - The commit message to use for the commit
  - handles - An array of ids from _gitfile_file_ or _gitfile_symlink_ resources which should be included in this commit

Outputs:

  - commit_message - The commit message for the commit that will be made
  - checkout_dir - The path to the git checkout input
  - file - The file(s) committed to.

# License

Apache2 - See the included LICENSE file for more details.

