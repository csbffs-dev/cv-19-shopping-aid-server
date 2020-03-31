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
	DateAndTimeSeenLayout = "Mon Jan 2, 2006 15:04:05 MST" // See https://golang.org/pkg/time/#Time.Format
)

type Item struct {
	Name         string         `datastore:"name"`
	StockReports []*StockReport `datastore:"stock_report"`
}

type QueryItemsReq struct {
	UserID string `json:"user_id"`
}

type QueryItemsResp struct {
	Items []*ItemInfo `json:"items"`
}

type ItemInfo struct {
	ItemName        string   `json:"item_name"`
	DateAndTimeSeen string   `json:"date_and_time_seen"`
	StoreName       string   `json:"store_name"`
	StoreAddr       *Address `json:"store_address"`
	UserInitials    string   `json:"user_initials"`
	UserZipCode     string   `json:"user_zip_code"`
	InStock         bool     `json:"in_stock"`
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

	resp := &QueryItemsResp{}
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

func parseItemIntoResp(item *Item, resp *QueryItemsResp) {
	for _, stockReport := range item.StockReports {
		itemInfo := &ItemInfo{
			ItemName:        item.Name,
			DateAndTimeSeen: time.Unix(stockReport.TimestampSec, 0).Format(DateAndTimeSeenLayout),
			StoreName:       stockReport.StoreInfo.Name,
			StoreAddr:       stockReport.StoreInfo.Addr,
			UserInitials:    fmt.Sprintf("%s.%s.", string(stockReport.UserInfo.FirstName[0]), string(stockReport.UserInfo.LastName[0])),
			UserZipCode:     stockReport.UserInfo.ZipCode,
			InStock:         stockReport.InStock,
		}
		resp.Items = append(resp.Items, itemInfo)
	}
}
