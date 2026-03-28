// Command workflow-plugin-marketplace is a workflow engine external plugin that
// provides marketplace pipeline step types for searching, installing, and
// managing workflow plugins.
package main

import (
	"github.com/GoCodeAlone/workflow-plugin-marketplace/internal"
	sdk "github.com/GoCodeAlone/workflow/plugin/external/sdk"
)

func main() {
	sdk.Serve(internal.NewMarketplacePlugin())
}
