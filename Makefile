.DEFAULT_GOAL := help
MAKEFLAGS += --silent --no-print-directory

BIN_DIR := ./bin
SCRIPTS_DIR := ./scripts
APP_NAME := x-repo-name
LDFLAGS += -s -w

# Print Makefile target step description for check.
# Only print 'check' steps this way, and not dependent steps, like 'install'.
# ${1} - step description
define _print_step
	printf -- '------\n%s...\n' "${1}"
endef

## Activate developer environment using devbox. Run `make install/devbox` first If you don't have devbox installed.
activate:
	devbox shell

## Install devbox binary.
install/devbox:
	$(call _print_step,Installing devbox)
	curl -fsSL https://get.jetpack.io/devbox | bash

.PHONY: build
## Build x-repo-name binary.
build:
	$(call _print_step,Building binary)
	mkdir -p $(BIN_DIR)
	go build -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/$(APP_NAME) ./cmd/$(APP_NAME)

.PHONY: release
## Build and release the binaries.
release:
	$(call _print_step,Releasing binary)
	@goreleaser release --snapshot --clean

.PHONY: test
## Run all unit tests.
test:
	$(call _print_step,Running unit tests)
	go test -race -cover ./...

.PHONY: test/coverage
## Produce test coverage report and inspect it in browser.
test/coverage:
	$(call _print_step,Running test coverage report)
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

.PHONY: check check/vet check/lint check/gosec check/spell check/trailing check/markdown check/format check/generate check/vulnerabilities
## Run all checks.
check: check/vet check/lint check/gosec check/spell check/trailing check/markdown check/format check/generate check/vulnerabilities

## Run 'go vet' on the whole project.
check/vet:
	$(call _print_step,Running go vet)
	go vet ./...

## Run golangci-lint all-in-one linter with configuration defined inside .golangci.yml.
check/lint:
	$(call _print_step,Running golangci-lint)
	golangci-lint run

## Check for security problems using gosec, which inspects the Go code by scanning the AST.
check/gosec:
	$(call _print_step,Running gosec)
	gosec -exclude-generated -quiet ./...

## Check spelling, rules are defined in cspell.json.
check/spell:
	$(call _print_step,Verifying spelling)
	yarn --silent cspell --no-progress '**/**'

## Check for trailing whitespaces in any of the projects' files.
check/trailing:
	$(call _print_step,Looking for trailing whitespaces)
	$(SCRIPTS_DIR)/check-trailing-whitespaces.bash

## Check markdown files for potential issues with markdownlint.
check/markdown:
	$(call _print_step,Verifying Markdown files)
	yarn --silent markdownlint '**/*.md' --ignore 'node_modules'

## Check for potential vulnerabilities across all Go dependencies.
check/vulnerabilities:
	$(call _print_step,Running govulncheck)
	govulncheck ./...

## Verify if the auto generated code has been committed.
check/generate:
	$(call _print_step,Checking if generated code matches the provided definitions)
	$(SCRIPTS_DIR)/check-generate.sh

## Verify if the files are formatted.
## You must first commit the changes, otherwise it won't detect the diffs.
check/format:
	$(call _print_step,Checking if files are formatted)
	$(SCRIPTS_DIR)/check-formatting.sh

.PHONY: generate generate/code
## Auto generate files.
generate: generate/code

## Generate Golang code.
generate/code:
	echo "Generating Go code..."
	go generate ./...

.PHONY: format format/go format/cspell
## Format files.
format: format/go format/cspell

## Format Go files.
format/go:
	$(call _print_step,Formatting Go files)
	gofumpt -l -w -extra .
	goimports -local=$$(head -1 go.mod | awk '{print $$2}') -w .
	golines -m 120 --ignore-generated --reformat-tags -w .

## Format cspell config file.
format/cspell:
	$(call _print_step,Formatting cspell.json configuration (words list))
	yarn --silent format-cspell-config

.PHONY: install
## Install all dev dependencies.
install: install/yarn

## Install JavaScript dependencies with yarn.
install/yarn:
	$(call _print_step,Installing yarn dependencies)
	yarn --silent install

.PHONY: help
## Print this help message.
help:
	$(SCRIPTS_DIR)/makefile-help.awk $(MAKEFILE_LIST)
