.PHONY: all fmt clean package test itest_%

all: fmt test

fmt:
	go fmt terraform-provider-gitfile/gitfile
	go fmt terraform-provider-gitfile

clean:
	make -C yelppack clean

itest_%:
	make -C yelppack $@

package: itest_lucid

test:
	go test terraform-provider-gitfile/gitfile
	go test terraform-provider-gitfile
