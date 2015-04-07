#!/bin/bash

project=$1
version=$2
iteration=$3

go get ${project}
mkdir /dist && cd /dist
fpm -s dir -t deb --name ${project} \
    --iteration ${iteration} --version ${version} \
    /go/bin/${project}=/usr/bin/
