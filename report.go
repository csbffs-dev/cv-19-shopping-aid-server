package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/datastore"
)

// StockReport represents the report entity. It is NOT stored as an entity in storage. Rather it is stored as a field of the item entity.
type StockReport struct {
	UsersInfo    []*User `datastore:"user_info"`
	StoreInfo    *Store  `datastore:"store_info"`
	TimestampSec int64   `datastore:"timestamp_sec"`
	InStock      bool    `datastore:"in_stock"`
	SeenCnt      int     `datastore:"seen_cnt"`
}

// ******************************************
// ** Begin UploadReport
// ******************************************

type UploadReportReq struct {
	UserID   string   `json:"user_id"`
	StoreID  string   `json:"store_id"`
	InStock  []string `json:"in_stock_items"`
	OutStock []string `json:"out_stock_items"`
}

// UploadReport updates each item in the in-stock list and out-stock list in the request
// with the stock report data.
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

	client, err := StorageClient(ctx)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer client.Close()

	if err := handleUploadToItems(ctx, client, store, user, req.InStock, true); err != nil {
		return http.StatusInternalServerError, err
	}

	if err := handleUploadToItems(ctx, client, store, user, req.OutStock, false); err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}

func handleUploadToItems(ctx context.Context, client *datastore.Client, store *Store, user *User, itemNames []string, checkInStock bool) error {
	now := time.Now().Unix()
	errFreq := 0
	var errResult error

	// For each item in itemNames, update item using name as key from storage. If item doesn't exist, create item
	// in storage.
	for _, itemName := range itemNames {
		// RunInTransaction guarantees that the get-then-put datastore operation is atomic.
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
			// Iterate through the item's stock reports to see if there is already one for the same
			// store. If so, just increment the seen count and timestamp rather than creating an entirely new report.
			for _, sr := range item.StockReports {
				if sr.StoreInfo.StoreID == store.StoreID && sr.InStock == checkInStock {
					// However, if it's the same user reporting it, do not increment the seenCnt.
					userAlreadyReported := false
					for _, u := range sr.UsersInfo {
						if u.UserID == user.UserID {
							userAlreadyReported = true
							break
						}
					}
					if !userAlreadyReported {
						sr.SeenCnt++
						sr.UsersInfo = append(sr.UsersInfo, &User{UserID: user.UserID, TimestampSec: now})
					}
					sr.TimestampSec = now
					if _, err := client.Put(ctx, key, &item); err != nil {
						return fmt.Errorf("failed to update item %q in storage with an existing stock report %v: %v", itemName, sr, err)
					}
					return nil
				}
			}
			sr := &StockReport{
				UsersInfo:    []*User{{UserID: user.UserID, TimestampSec: now}},
				StoreInfo:    store,
				TimestampSec: now,
				InStock:      checkInStock,
				SeenCnt:      1,
			}
			item.StockReports = append(item.StockReports, sr)
			if _, err := client.Put(ctx, key, &item); err != nil {
				return fmt.Errorf("failed to update item %q in storage with new stock report %v: %v", itemName, sr, err)
			}
			return nil
		}); err != nil {
			// Rather than returning an error once a transaction fails, try to run all transactions for items
			// and report the first error and number of errors at the end.
			errFreq++
			if errResult == nil {
				errResult = err
			}
		}
	}

	if errResult != nil {
		return fmt.Errorf("Encountered %d failures, recorded the first one: %v", errFreq, errResult)
	}
	return nil
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
			return fmt.Errorf("out-of-stock item at index %d is empty", i)
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

// ******************************************
// ** END UploadReport
// ******************************************
