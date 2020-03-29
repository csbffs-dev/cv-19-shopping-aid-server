package main

import (
	"context"
	"fmt"
	"net/http"
)

type Item struct {
	Name            string
	InStockReports  []*StockReport
	OutStockReports []*StockReport
}

// QueryItems fetches the list of items in storage.
func QueryItems(ctx context.Context, w http.ResponseWriter, r *http.Request) (int, error) {
	return http.StatusNotImplemented, fmt.Errorf("not supported yet")
}
