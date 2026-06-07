// Command official is the offline frontend that generates the Official UniFi
// OpenAPI models (unifi/official/models.generated.go) from the committed spec
// snapshot. Stage 3 folds this into the main generator's second pass; run it
// standalone as: go run . (from codegen/official).
package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	openapiDir := flag.String("openapi-dir", "../openapi", "directory holding the committed integration-<ver>.json snapshot")
	outDir := flag.String("out-dir", "../../unifi/official", "output directory for the generated Official surface files")
	pkg := flag.String("package", defaultPackageName, "package name for the generated code")
	flag.Parse()

	spec, err := ResolveSnapshot(*openapiDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	if err := GenerateAll(spec, *outDir, *pkg); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
