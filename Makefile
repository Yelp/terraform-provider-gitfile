.PHONY: all fmt .git/hooks/pre-commit clean package test itest_%

all: fmt .git/hooks/pre-commit test

fmt:
	go fmt ./...

clean:
	make -C yelppack clean

itest_%:
	make -C yelppack $@

package: itest_lucid

test:
	go test -v ./gitfile/...

.git/hooks/pre-commit:
	if [ ! -f .git/hooks/pre-commit ]; then ln -s ../../git-hooks/pre-commit .git/hooks/pre-commit; fi

