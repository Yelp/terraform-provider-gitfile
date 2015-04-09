#!/bin/bash
set -ex
rm -rf example.git checkout terraform.tfstate terraform.tfstate.backup
mkdir example.git
cd example.git
git init
touch .exists
git add .exists
git commit -m"Initial commit"
git checkout -b move_HEAD
cd ..
terraform apply
terraform apply
cd checkout
git log | grep 'Created by terraform gitfile_commit'
git fetch
git log origin/master | grep 'Created by terraform gitfile_commit'
if [ ! -f terraform ]; then
    exit 1
fi
