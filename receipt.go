package main

import (
	"fmt"
	"net/http"
)

// ParseReceipt parses a receipt for the list of items.
func ParseReceipt(r *http.Request) (int, error) {
	return http.StatusNotImplemented, fmt.Errorf("not supported yet")
}
