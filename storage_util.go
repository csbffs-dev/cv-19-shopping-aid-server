package main

import (
	"context"
	"fmt"
	"os"

	"cloud.google.com/go/datastore"
)

const (
	UserKind  = "User"
	StoreKind = "Store"
	ItemKind  = "Item"
)

// StorageClient returns a storage client instance.
func StorageClient(ctx context.Context) (*datastore.Client, error) {
	// TODO: Reuse storage client for all calls rather than invoking it for each one.
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
