package main

import (
	"context"
	"fmt"
	"net/http"
)

// UploadReport uploads a report to storage.
func UploadReport(ctx context.Context, w http.ResponseWriter, r *http.Request) (int, error) {
	return http.StatusNotImplemented, fmt.Errorf("not supported yet")
}
