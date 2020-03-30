package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/datastore"
)

type StockReport struct {
	UserInfo     *User  `datastore:"user_info"`
	StoreInfo    *Store `datastore:"store_info"`
	TimestampSec int64  `datastore:"timestamp_sec"`
	InStock      bool   `datastore:"in_stock"`
}

type UploadReportReq struct {
	UserID   string   `json:"user_id"`
	StoreID  string   `json:"store_id"`
	InStock  []string `json:"in_stock_items"`
	OutStock []string `json:"out_stock_items"`
}

// UploadReport uploads a report to storage.
func UploadReport(ctx context.Context, w http.ResponseWriter, r *http.Request) (int, error) {
	var req UploadReportReq
	if err := DecodeReq(r.Body, &req); err != nil {
		return http.StatusBadRequest, err
	}
	if err := cleanAndValidateUploadReportReq(&req); err != nil {
		return http.StatusBadRequest, err
	}

	user, ok, err := GetUserInStorage(ctx, req.UserID)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to check user creds: %v", err)
	}
	if !ok {
		return http.StatusForbidden, fmt.Errorf("user id is invalid: %q", req.UserID)
	}

	store, err := GetStoreInStorage(ctx, req.StoreID)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	// for each item in inStock, fetch item using name as key from storage.
	// if item doesn't exist, create item in storage
	client, err := StorageClient(ctx)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer client.Close()

	now := time.Now().Unix()
	for _, itemName := range req.InStock {
		if _, err := client.RunInTransaction(ctx, func(tx *datastore.Transaction) error {
			var item Item
			key := datastore.NameKey(ItemKind, itemName, nil)
			if err := client.Get(ctx, key, &item); err != nil {
				if err != datastore.ErrNoSuchEntity {
					return fmt.Errorf("failed to fetch item %q from storage: %v", itemName, err)
				}
				item.Name = itemName
				item.StockReports = make([]*StockReport, 0)
			}
			item.StockReports = append(item.StockReports, &StockReport{
				UserInfo:     user,
				StoreInfo:    store,
				TimestampSec: now,
				InStock:      true,
			})
			if _, err := client.Put(ctx, key, &item); err != nil {
				return fmt.Errorf("failed to update item %q in storage: %v", itemName, err)
			}
			return nil
		}); err != nil {
			return http.StatusInternalServerError, err
		}
	}
	for _, itemName := range req.OutStock {
		if _, err := client.RunInTransaction(ctx, func(tx *datastore.Transaction) error {
			var item Item
			key := datastore.NameKey(ItemKind, itemName, nil)
			if err := client.Get(ctx, key, &item); err != nil {
				if err != datastore.ErrNoSuchEntity {
					return fmt.Errorf("failed to fetch item %q from storage: %v", itemName, err)
				}
				item.Name = itemName
				item.StockReports = make([]*StockReport, 0)
			}
			item.StockReports = append(item.StockReports, &StockReport{
				UserInfo:     user,
				StoreInfo:    store,
				TimestampSec: now,
				InStock:      false,
			})
			if _, err := client.Put(ctx, key, &item); err != nil {
				return fmt.Errorf("failed to update item %q in storage: %v", itemName, err)
			}
			return nil
		}); err != nil {
			return http.StatusInternalServerError, err
		}
	}

	return http.StatusOK, nil
}

func cleanAndValidateUploadReportReq(req *UploadReportReq) error {
	if req.UserID == "" {
		return fmt.Errorf("missing user id")
	}
	if req.StoreID == "" {
		return fmt.Errorf("missing store id")
	}
	if len(req.InStock) == 0 && len(req.OutStock) == 0 {
		return fmt.Errorf("in-stock and out-of-stock items are both empty")
	}
	// An edge case is if the same item appears multiple times in the inStock array,
	// in the outStock array, and/or in both arrays. Prune duplicates in each array.
	// In case of both arrays, we bias the item in the inStock array. It will not
	// appear in the outStock array.
	seen := make(map[string]bool)
	inStock := make([]string, 0)
	outStock := make([]string, 0)
	for i := range req.InStock {
		item := strings.ToLower(strings.TrimSpace(req.InStock[i]))
		if item == "" {
			return fmt.Errorf("in-stock item at index %d is empty", i)
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = true
		inStock = append(inStock, item)
	}
	for i := range req.OutStock {
		item := strings.ToLower(strings.TrimSpace(req.OutStock[i]))
		if item == "" {
			return fmt.Errorf("out-of-stock item at index %d is empty, i")
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = true
		outStock = append(outStock, item)
	}
	req.InStock = inStock
	req.OutStock = outStock
	return nil
}
