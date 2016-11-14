#!/bin/bash

project=$1
version=$2
iteration=$3

go get github.com/Yelp/${project}
mkdir /dist && cd /dist
fpm -s dir -t deb --deb-no-default-config-files --name ${project} \
    --iteration ${iteration} --version ${version} \
    /go/bin/${project}=/nail/opt/terraform-0.7/bin/
