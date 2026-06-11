# File Uploads in go-unifi

This document describes how to use the file upload functionality in the go-unifi client.

## Overview

The go-unifi client provides two methods for uploading portal files to the UniFi controller:

1. `UploadPortalFile` - Upload a portal file from a file path on disk
2. `UploadPortalFileFromReader` - Upload a portal file from an `io.Reader` (e.g., from memory, network stream, etc.)

Both methods use the `multipart/form-data` format for file uploads, which is required by the UniFi controller.

## Examples

### Uploading a file from disk

```go
package main

import (
	"context"
	"log"

	"github.com/filipowm/go-unifi/v2/unifi"
)

func main() {
	// Create a client
	client, err := unifi.NewClient(&unifi.ClientConfig{
		URL:    "https://your-unifi-controller:8443",
		APIKey: "your-api-key",
	})
	if err != nil {
		log.Fatalf("Error creating client: %v", err)
	}

	ctx := context.Background()

	// Upload the portal file to the controller for the "default" site
	portalFile, err := client.UploadPortalFile(ctx, "default", "/path/to/your/file.png")
	if err != nil {
		log.Fatalf("Error uploading file: %v", err)
	}

	log.Printf("Upload successful: id=%s url=%s", portalFile.ID, portalFile.URL)
}
```

### Uploading a file from memory

```go
package main

import (
	"bytes"
	"context"
	"log"

	"github.com/filipowm/go-unifi/v2/unifi"
)

func main() {
	// Create a client
	client, err := unifi.NewClient(&unifi.ClientConfig{
		URL:    "https://your-unifi-controller:8443",
		APIKey: "your-api-key",
	})
	if err != nil {
		log.Fatalf("Error creating client: %v", err)
	}

	ctx := context.Background()

	// Create file content in memory
	fileContent := []byte("...image or HTML content...")
	reader := bytes.NewReader(fileContent)

	// Upload the portal file from the reader for the "default" site
	portalFile, err := client.UploadPortalFileFromReader(ctx, "default", reader, "myfile.png")
	if err != nil {
		log.Fatalf("Error uploading file: %v", err)
	}

	log.Printf("Upload successful: id=%s url=%s", portalFile.ID, portalFile.URL)
}
```

## API Reference

### UploadPortalFile

```go
func (c *client) UploadPortalFile(ctx context.Context, site string, filepath string) (*PortalFile, error)
```

Uploads a portal file to the UniFi controller from a file path on disk.

Parameters:
- `ctx`: The context for the request
- `site`: The site name (e.g. `"default"`)
- `filepath`: Path to the file on disk

Returns the uploaded `*PortalFile` (with `ID`, `URL`, `Filename`, etc.) or an error.

### UploadPortalFileFromReader

```go
func (c *client) UploadPortalFileFromReader(ctx context.Context, site string, reader io.Reader, filename string) (*PortalFile, error)
```

Uploads a portal file to the UniFi controller from an `io.Reader`.

Parameters:
- `ctx`: The context for the request
- `site`: The site name (e.g. `"default"`)
- `reader`: Reader with the file content
- `filename`: Name of the file to use in the upload

Returns the uploaded `*PortalFile` or an error.

## Notes

- These methods use `POST` requests for file uploads under the portal-file endpoint.
- The content type for the request is automatically detected from the file content.
- All existing client features like interceptors, error handling, and request validation are preserved.
- For lower-level file uploads to custom endpoints, use the internal `UploadFile`/`UploadFileFromReader`
  methods via the raw-call surface (`c.Do`/`c.Post`), but note these are not part of the public `Client`
  interface.
