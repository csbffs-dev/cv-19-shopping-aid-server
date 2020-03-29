package main

import (
	"context"
	"fmt"
	"net/http"
)

// ParseReceipt parses a receipt for the list of items.
func ParseReceipt(ctx context.Context, w http.ResponseWriter, r *http.Request) (int, error) {
	return http.StatusNotImplemented, fmt.Errorf("not supported yet")
}
