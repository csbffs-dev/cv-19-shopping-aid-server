package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"cloud.google.com/go/datastore"
	"google.golang.org/api/iterator"
)

const (
	DateAndTimeSeenLayout = "Mon Jan 2 15:04:05" // See https://golang.org/pkg/time/#Time.Format
)

type Item struct {
	Name         string         `datastore:"name"`
	StockReports []*StockReport `datastore:"stock_report"`
}

// ******************************************
// ** Begin QueryItems
// ******************************************

type QueryItemsReq struct {
	UserID string `json:"user_id"`
}

type QueryItemsResp struct {
	Items []*ItemInfo `json:"items"`
}

// TODO: Change DateAndTimeSeen to HoursAgo or DaysAgo
type ItemInfo struct {
	ItemName        string `json:"item_name"`
	DateAndTimeSeen string `json:"date_and_time_seen"`
	StoreName       string `json:"store_name"`
	StoreAddr       string `json:"store_address"`
	InStock         bool   `json:"in_stock"`
}

// QueryItems fetches the list of items in storage.
func QueryItems(ctx context.Context, w http.ResponseWriter, r *http.Request) (int, error) {
	var req QueryItemsReq
	if err := DecodeReq(r.Body, &req); err != nil {
		return http.StatusBadRequest, err
	}

	if err := validateQueryItemsReq(req); err != nil {
		return http.StatusBadRequest, err
	}

	_, ok, err := GetUserInStorage(ctx, req.UserID)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to check user creds: %v", err)
	}
	if !ok {
		return http.StatusForbidden, fmt.Errorf("user id is invalid: %q", req.UserID)
	}

	client, err := StorageClient(ctx)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	resp := &QueryItemsResp{Items: make([]*ItemInfo, 0)}
	q := datastore.NewQuery(ItemKind)
	it := client.Run(ctx, q)
	for {
		var t Item
		_, err := it.Next(&t)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return http.StatusInternalServerError, fmt.Errorf("failed to query for all items: %v", err)
		}
		// TODO: Filter response based on nearness to user's zipcode.
		parseItemIntoResp(&t, resp)
	}

	if err := EncodeResp(w, &resp); err != nil {
		return http.StatusInternalServerError, err
	}
	return http.StatusOK, nil
}

func validateQueryItemsReq(req QueryItemsReq) error {
	if req.UserID == "" {
		return fmt.Errorf("missing user id")
	}
	return nil
}

// ******************************************
// ** END QueryItems
// ******************************************

func parseItemIntoResp(item *Item, resp *QueryItemsResp) {
	for _, stockReport := range item.StockReports {
		itemInfo := &ItemInfo{
			ItemName:        item.Name,
			DateAndTimeSeen: time.Unix(stockReport.TimestampSec, 0).Format(DateAndTimeSeenLayout),
			StoreName:       stockReport.StoreInfo.Name,
			StoreAddr:       stockReport.StoreInfo.Addr,
			InStock:         stockReport.InStock,
		}
		resp.Items = append(resp.Items, itemInfo)
	}
}
