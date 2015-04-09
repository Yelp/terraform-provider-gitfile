.PHONY: all clean test

all: test

clean:
	rm -rf */example.git */checkout */terraform.tfstate.backup */terraform.tfstate

test: clean
	for i in $$(find . -name test_\* -maxdepth 1 -type d); do cd $$i; ./test.sh; cd ..; done

