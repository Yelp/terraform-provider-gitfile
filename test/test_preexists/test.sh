#!/bin/bash
set -ex
rm -rf example.git checkout terraform.tfstate terraform.tfstate.backup
mkdir example.git
cd example.git
git init
touch .exists
git add .exists
git commit -m"Initial commit"
echo 'preexisting_commits' > terraform
git add terraform
git commit -m"PRE"
git checkout -b move_HEAD
cd ..
terraform apply
terraform apply
cd checkout
git fetch
# We did do a commit
git log origin/master | grep 'Created by terraform gitfile_commit'
# But it has no diff
if [ "$(git diff HEAD~1..HEAD | wc -l | awk '{ print $1 }')" != "0" ];then
    exit 1
fi
if [ ! -f terraform ]; then
    exit 1
fi
