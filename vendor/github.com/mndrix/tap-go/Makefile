TESTS = auto check diagnostic failing known skip todo writer yaml
GOPATH ?= $(CURDIR)/gopath

.PHONY: $(TESTS)

all: $(foreach t,$(TESTS),test/$(t)/test)
	prove -v -e '' test/*/test

clean:
	rm -f test/*/test

test/%/test: test/%/*.go tap.go yaml_json.go yaml_yaml.go
	go build -o $@ -tags yaml ./test/$*

$(TESTS): %: test/%/test
	prove -v -e '' test/$@/test
