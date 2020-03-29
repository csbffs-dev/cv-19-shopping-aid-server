package main

import (
	"fmt"
	"net/http"
)

// QueryStores fetches the list of stores in storage.
func QueryStores(r *http.Request) (int, error) {
	return http.StatusNotImplemented, fmt.Errorf("not supported yet")
}
