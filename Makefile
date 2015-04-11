.PHONY: all fmt .git/hooks/pre-commit terraform-provider-gitfile clean package test itest_%

all: fmt .git/hooks/pre-commit test terraform-provider-gitfile

fmt:
	go fmt ./...

clean:
	make -C yelppack clean
	rm -f terraform-provider-gitfile
	rm -rf test/example.git test/checkout test/terraform.tfstate.backup test/terraform.tfstate

terraform-provider-gitfile: test
	go build

dev: terraform-provider-gitfile
	cp terraform-provider-gitfile $$(echo $$GOPATH|sed -e's/://')/bin

integration: dev
	make -C test

itest_%:
	make -C yelppack $@

package: itest_lucid

test:
	go test -v ./gitfile/...

.git/hooks/pre-commit:
	if [ ! -f .git/hooks/pre-commit ]; then ln -s ../../git-hooks/pre-commit .git/hooks/pre-commit; fi

