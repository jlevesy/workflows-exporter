RELEASE_TAG ?= dev
TIMEOUT ?= 5s

### Tests

.PHONY: test
test: unit_test

.PHONY: unit_test
unit_test:
	go test -v -cover -race -timeout=$(TIMEOUT) ./...

### CI

.PHONY: ci_release
ci_release: ci_create_release ci_push_image

.PHONY: ci_create_release
ci_create_release:
	gh release create $(RELEASE_TAG) --generate-notes

.PHONY: ci_push_image
ci_push_image:
	ko publish --bare -t $(RELEASE_TAG) ./cmd/exporter
