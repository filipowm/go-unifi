// Package internal contains the Internal-API code generation engine. The root
// codegen (package main) orchestrates version resolution, downloading, and the
// Official-API pass; this package owns the Internal-API generation pipeline
// (resources, clients).
package internal

import (
	"fmt"

	"github.com/filipowm/go-unifi/codegen/shared"
)

// Generate runs the Internal-API code generation pass: builds resources from
// the downloaded controller field-definition JSONs (fieldsDir) and the hand-
// maintained V2 field definitions (v2BaseDir), and emits <resource>.generated.go
// and client.generated.go into outDir. It does NOT write version.generated.go —
// root's writeVersionArtifacts owns that file.
//
// floorFieldsDir, when non-empty, is the supported-version floor's field-JSON
// dir: the resource set is generated as a floor ∪ fieldsDir merge (newest wins),
// so resources retired before the floor are dropped while newest field shapes
// apply on top of the floor. An empty floorFieldsDir disables the merge.
//
// This is the entry point called by root main.go's generate() after downloading
// the controller package. The Official-API pass (codegen/official) is a separate
// standalone tool invoked by root's runOfficialPass(); it is out of scope here.
func Generate(floorFieldsDir, fieldsDir, v2BaseDir, outDir string, customizer CodeCustomizer, logger shared.Logger) error {
	logger = orDefaultLogger(logger)
	customizer.logger = logger

	logger.Infoln("Generating resources code...")
	if err := generateCode(floorFieldsDir, fieldsDir, v2BaseDir, outDir, customizer, logger); err != nil {
		return fmt.Errorf("unable to generate resources code: %w", err)
	}

	return nil
}
