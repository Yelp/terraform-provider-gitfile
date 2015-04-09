# terraform-provider-gitfile

## Synopsis

A [Terraform](http://terraform.io) plugin to commit files to git repositories.

This allows you to export terraform managed state into other systems which are controlled
by git repositories - for example commit server IPs to DNS config repositories,
or write out hiera data into your puppet configuration.

## Example:

    resource "gitfile_checkout" "puppet" {
        repo = "git@yourcompany.com:bind"
        branch = "master"
        path = "${path.root}/../other_git_checkouts/bind"
    }

    resource "aws_instance" "importantbox" {
        ....
    }

    resource "gitfile_commit" "importantbox A record" {
        checkout_dir = "${gitfile_checkout.bind.path}"
        file {
            path = "zones.internal/${var.region}-terraform.fragment"
            contents = "importantbox A ${aws_instance.importantbox.private_ip}"
        }
    }

## Resources

### gitfile_checkout

Checks out a git repository onto your local filesystem from within a terraform provider.

This is mostly used to ensure that a checkout is present, before using the _gitfile_commit_
resource to commit some Terraform generated data.

Inputs:
    * repo - The git path to the repository, this can be anything you can feed to 'git clone'
    * branch - The branch to checkout, defaults to 'master'
    * path - The file path on filesystem for where to put the checkout

Outputs:
    * path - The file path on filesystem where the repository has been checked out

### gitfile_commit

Makes a git commit of a specified file (in an already checked out repository)
with specified contents, and pushes the branch checked out to origin.

Note that even if the a file with the same contents Terraform creates already exists,
Terraform will create an empty commit with the specified commit message.

Inputs:
    * checkout_dir - The path to a git checkout, this can have been made by _gitfile_checkout_ or any other mechanism.
    * file {} - Files to be committed, this block can be repeated multiple times. Each block contains:
        * path - The path (within the repository) of the file to create
        * contents - The contents to set the file to

Outputs:
    * commit_message - The commit message for the commit that will be made
    * checkout_dir - The path to the git checkout input
    * file - The file(s) committed to.

# License

Apache2 - See the included LICENSE file for more details.

