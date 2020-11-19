CI                  := $(if ${CI},${CI},0)
VERSION             := $(if ${CDS_SEMVER},${CDS_SEMVER},snapshot)
CDS_VERSION         := $(if ${CDS_VERSION},${CDS_VERSION},snapshot)
UNAME               := $(shell uname)
UNAME_LOWERCASE     := $(shell uname -s| tr A-Z a-z)

.PHONY: help
help:
	@grep -hE '^[ a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-17s\033[0m %s\n", $$1, $$2}'

