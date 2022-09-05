//go:build tools

package tools

import (
	_ "github.com/caarlos0/svu"
	_ "github.com/golangci/golangci-lint/cmd/golangci-lint"
	_ "github.com/goreleaser/goreleaser"
	_ "golang.org/x/vuln/cmd/govulncheck"
)
