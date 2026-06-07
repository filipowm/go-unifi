package unifi

// This will generate the *.generated.go files in this package for the specified
// client controller version. The -floor-version bounds the resource set below by
// the supported floor (9.0.114): resources retired before it are dropped, while
// the 9.5.21 snapshot supplies the newest field shapes.
//go:generate go run ../codegen/ -version-base-dir=../codegen/ -official-spec-version=10.1.78 -floor-version=9.0.114 9.5.21
