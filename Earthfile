VERSION 0.6
FROM golang:1.19-alpine
ARG BINPATH=/usr/local/bin/
ARG GOCACHE=/go-cache

deps:
    WORKDIR /src
    ENV GO111MODULE=on
    ENV CGO_ENABLED=0
    COPY go.mod go.sum ./
    RUN go mod download

build:
    FROM +deps
    COPY --dir pkg/ cmd/ .
    ARG GOOS=linux
    ARG GOARCH=amd64
    ARG VARIANT
    RUN --mount=type=cache,target=$GOCACHE \
        GOARM=${VARIANT#"v"} go build -ldflags="-w -s" -o out/ ./...
    SAVE ARTIFACT out/*

test:
    FROM +deps
    COPY --dir pkg/ cmd/ .
    RUN --mount=type=cache,target=$GOCACHE \
        go test ./...

lint-go:
    FROM +deps
    COPY +tools/golangci-lint $BINPATH
    COPY --dir pkg/ cmd/ .golangci.yml .
    ARG GOLANGCI_LINT_CACHE=/golangci-cache
    RUN --mount=type=cache,target=$GOCACHE \
        --mount=type=cache,target=$GOLANGCI_LINT_CACHE \
        golangci-lint run -v ./...

lint-commit:
    FROM node:alpine
    RUN apk --no-cache --update add git
    RUN npm install -g @commitlint/cli @commitlint/config-conventional
    WORKDIR /src
    COPY --dir .git/ .
    # check all commits to be in the right format
    RUN commitlint --to HEAD --verbose -x @commitlint/config-conventional

lint-vulns:
    FROM +tools
    COPY --dir pkg/ cmd/ go.mod go.sum .
    RUN govulncheck ./...

lint:
    BUILD +lint-go
    BUILD +lint-commit
    BUILD +lint-vulns
    BUILD +build-test
    BUILD +test

generate:
    COPY +build/docgen $BINPATH
    RUN mkdir -p docs && docgen -o docs
    SAVE ARTIFACT docs AS LOCAL docs/cmd

tag-release:
    FROM +tools
    RUN apk add --no-cache --update openssh
    RUN mkdir -p -m 0700 ~/.ssh && ssh-keyscan github.com >> ~/.ssh/known_hosts
    COPY --dir .git/ .
    RUN --ssh git checkout -q main && git fetch && git tag $(svu next)
    RUN --push --ssh git push --tags
    SAVE ARTIFACT .git/refs/tags AS LOCAL .git/refs/tags

release:
    FROM +tools
    COPY --dir .git/ pkg/ cmd/ .goreleaser.yaml .
    RUN --push \
        --secret GITHUB_TOKEN \
        --mount=type=cache,target=$GOCACHE \
        goreleaser release --clean --skip-validate

tools:
    FROM +deps
    # version from tools.go
    ARG GOBIN=/go/bin
    RUN go install \
        golang.org/x/vuln/cmd/govulncheck \
        github.com/golangci/golangci-lint/cmd/golangci-lint \
        github.com/caarlos0/svu \
        github.com/goreleaser/goreleaser
    RUN apk add --no-cache --update git
    SAVE ARTIFACT $GOBIN/*
