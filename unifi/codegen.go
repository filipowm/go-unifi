package unifi

// 10.2.97 --> 10.3.58 --> 10.4.57 --> 10.5.54
// This will generate the *.generated.go files in this package for the specified
// client controller version. The positional argument is the Official-API spec
// version; -legacy-version pins the Internal/legacy controller resources.
//go:generate go run ../codegen/ -version-base-dir=../codegen/ -legacy-version=9.5.21 10.2.97
