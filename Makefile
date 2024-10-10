MAKEFLAGS+=--no-print-directory

.PHONY: ci
ci: test

.PHONY: test
test:
	@go test ./...

.PHONY: watch
watch:
	@trap exit SIGINT; while true; do \
	  git ls-files -c -o '*.go' | entr -r -d -c $(MAKE) test; \
	done

.PHONY: build
build:
	@go build -o ./build/tektor main.go
