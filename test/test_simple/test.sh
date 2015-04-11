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
gitfile_checkout_path="$(terraform output gitfile_checkout_path)"
if [ "$gitfile_checkout_path" != "checkout" ];then
    exit 1
fi
gitfile_commit_commit_message="$(terraform output gitfile_commit_commit_message)"
if [ "$gitfile_commit_commit_message" != "Created by terraform gitfile_commit" ];then
    exit 1
fi
gitfile_commit_checkout_dir="$(terraform output gitfile_commit_checkout_dir)"
if [ "$gitfile_commit_checkout_dir" != "checkout" ];then
    exit 1
fi

cd checkout
git log | grep 'Created by terraform gitfile_commit'
git fetch
git log origin/master | grep 'Created by terraform gitfile_commit'
if [ ! -f terraform ]; then
    exit 1
fi
