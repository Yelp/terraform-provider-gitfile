#!/bin/bash

set -eu

tf_version=$2

dpkg -i "$1"
echo "installed package"
test -x /nail/opt/terraform-${tf_version}/bin/terraform-provider-gitfile
