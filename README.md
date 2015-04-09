# terraform-provider-gitfile

A [Terraform](http://terraform.io) plugin to commit files to git repositories.

This allows you to export terraform managed state into other systems which are controlled
by git repositories - for example commit server IPs to DNS config repositories,
or write out hiera data into your puppet configuration.

## Example use:

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

# License

Apache2 - See the included LICENSE file for more details.

