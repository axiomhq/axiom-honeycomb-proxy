//go:build tools

package axiomhoneycombproxy

import (
	_ "github.com/golangci/golangci-lint/cmd/golangci-lint"
	_ "github.com/goreleaser/goreleaser/v2"
	_ "gotest.tools/gotestsum"
)
