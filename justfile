GO_FLAGS := "CGO_ENABLED=0"
TEST_FLAGS := "CGO_ENABLED=0 GOEXPERIMENT=synctest"
TEST_RACE_FLAGS := "CGO_ENABLED=1 GOEXPERIMENT=synctest"

# List available commands
default:
	@just --list

# Build
build:
	go build ./...

# Get git version
version:
	@just _get-git-version

# Get the next git version
next-version *ARGS:
	@just _get-git-version-next {{ARGS}}

# Create a release using next semantic version
release *ARGS:
	@just _create-release-next {{ARGS}}

# Generate go code
gen:
	go generate ./...

# Run tests
test:
	{{TEST_FLAGS}} go test -v ./...

fuzz:
    {{TEST_FLAGS}} go test -fuzz=Fuzz -fuzztime 60s

# Run tests showing coverage
test-coverage:
	{{TEST_FLAGS}} go test -cover -v ./...

# Generate test coverage report
coverage:
	{{TEST_FLAGS}} go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Run tests with race detector
test-race:
	{{TEST_RACE_FLAGS}} go test -race -v ./...

# Format go code
fmt:
	go fmt ./...

# Clean build artifacts
clean:
	rm -f coverage.out coverage.html
	go clean

# Vet go code
vet:
	go vet ./...

# Run linter
lint:
	golangci-lint run ./...

_get-git-version:
	@git describe --tags --always --match=v* 2>/dev/null || echo "dev"

[positional-arguments]
_get-git-version-next *ARGS:
	#!/usr/bin/env bash
	git fetch --all --tags &>/dev/null

	latest_tag=$(git tag --sort='v:refname' | tail -1)
	if [ -z "${latest_tag}" ]; then
		latest_tag="v0.0.0"
	fi
	latest_version=$(echo "${latest_tag}" | tr -d 'v')

	IFS="." read -r major minor patch <<<"${latest_version}"

	incr_type=${1:-patch}
	case "${incr_type}" in
		patch)
			patch=$((patch + 1))
			;;
		minor)
			minor=$((minor + 1))
			patch=0
			;;
		major)
			major=$((major + 1))
			minor=0
			patch=0
			;;
		*)
			echo >&2 "Error: Invalid bump type '${incr_type}'. Must be patch, minor, or major."
			exit 1
			;;
	esac

	next_v="${major}.${minor}.${patch}"

	if [[ ! "${next_v}" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
		echo >&2 "Error: Generated version ${next_v} is not valid semantic version."
		exit 1
	fi

	echo "${next_v}"

[positional-arguments]
_create-release-next *ARGS:
	#!/usr/bin/env bash
	bump="${1:-patch}"
	repo_root=$(git rev-parse --show-toplevel) || { echo >&2 "Error: Not a git repository."; exit 1; }

	cd "${repo_root}" || { echo >&2 "Error: Could not change to repository root."; exit 1; }

	next_v=$(just next-version "${bump}") || exit 1

	echo "Next version is: ${next_v}"

	if [ -z "${FORCE}" ]; then
		echo "Checking for unstaged and uncommitted changes..."
		git update-index -q --ignore-submodules --refresh

		if ! git diff-files --quiet --ignore-submodules -- || \
			! git diff-index --cached --quiet HEAD --ignore-submodules --; then
			echo >&2 "Cannot tag release: you have unstaged or uncommitted changes."
			git status --short >&2 # Show what's changed
			echo >&2 "Please commit or stash them."
			exit 1
		fi
	fi

	echo "Ensuring remote is up-to-date..."
	if ! git pull --ff-only; then
		echo >&2 "Error: Failed to pull latest changes. Please resolve and try again."
		exit 1
	fi

	echo "Pushing current branch..."
	if ! git push; then
		echo >&2 "Error: Failed to push current branch. Please resolve and try again."
		exit 1
	fi

	echo "Tagging release..."
	tag_name="v${next_v}"
	if git rev-parse "${tag_name}" &>/dev/null; then
		echo >&2 "Warning: Tag '${tag_name}' already exists locally. Skipping local tag creation."
	else
		if ! git tag "${tag_name}"; then
			echo >&2 "Error: Failed to create tag '${tag_name}'."
			exit 1
		fi
	fi

	echo "Pushing tags..."
	if ! git push --tags; then
		echo >&2 "Error: Failed to push tags."
		exit 1
	fi

	echo "Done. Tag ${tag_name} has been pushed."
