package main

import (
	"context"
	"fmt"
	"os"

	"cloud.google.com/go/datastore"
)

const (
	UserKind = "User"
	StoreKind = "Store"
)

// StorageClient returns a storage client instance.
func StorageClient(ctx context.Context) (*datastore.Client, error) {
	// TODO: 
	projectID := os.Getenv("PROJECT_ID") // See app.yaml
	if projectID == "" {
		return nil, fmt.Errorf("project id env variable is not set")
	}
	client, err := datastore.NewClient(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage client: %v", err)
	}
	return client, nil
}
