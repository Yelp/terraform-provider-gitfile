#!/bin/bash

set -eu

dpkg -i "$1"
test -x /nail/opt/terraform-0.7/bin/terraform-provider-gitfile
