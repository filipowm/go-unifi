package unifi

// This will generate the *.generated.go files in this package for the specified
// client controller version. The positional argument is the Official-API spec
// version; -legacy-version pins the Internal/legacy controller resources.
//go:generate go run ../codegen/ -version-base-dir=../codegen/ -legacy-version=9.5.21 latest
