#!/bin/bash

set -e

project=$1
version=$2
iteration=$3

cd /go/src/github.com/Yelp/terraform-provider-gitfile
go get
go test ./...
go build .
mkdir /dist && cd /dist
fpm -s dir -t deb --name ${project} \
    --iteration ${iteration} --version ${version} \
    --prefix /usr/bin/ \
    /go/bin/terraform-provider-gitfile

