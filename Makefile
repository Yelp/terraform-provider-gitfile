.PHONY: all fmt clean package test itest_%

all: fmt test

fmt:
	go fmt ./...

clean:
	make -C yelppack clean

itest_%:
	make -C yelppack $@

package: itest_lucid

test:
	go test -v ./gitfile/...

