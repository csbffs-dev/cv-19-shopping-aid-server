package main

import (
	"fmt"
	"net/http"
)

// QueryItems fetches the list of items in storage.
func QueryItems(r *http.Request) (int, error) {
	return http.StatusNotImplemented, fmt.Errorf("not supported yet")
}
