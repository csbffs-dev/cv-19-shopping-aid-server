package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/datastore"
	"google.golang.org/api/iterator"
)

const (
	secondsToHour = 3600
	secondsToDay  = 3600 * 24
)

type Item struct {
	Name         string         `datastore:"name"`
	StockReports []*StockReport `datastore:"stock_report"`
}

type Tokens []string

var itemNames []string
var itemTokens []Tokens

func init() {
	f, err := os.Open("./assets/itemsAndTokens.txt")
	if err != nil {
		log.Fatalf("failed to open items data file: %v", err)
	}
	scanner := bufio.NewScanner(f)
	scanner.Split(bufio.ScanLines)

	// Keep ordering of item token data
	for scanner.Scan() {
		data := strings.Split(scanner.Text(), ":")
		itemNames = append(itemNames, data[0])
		itemTokens = append(itemTokens, strings.Split(data[1], ","))
	}
	log.Println("successfully parsed item token data")
}

// ******************************************
// ** Begin QueryItemTokens
// ******************************************

type QueryItemTokensReq struct {
	UserID string `json:"user_id"`
}

type QueryItemTokensResp []*ItemTokenInfo

type ItemTokenInfo struct {
	Name   string   `json:"name"`
	Tokens []string `json:"tokens"`
}

func QueryItemTokens(ctx context.Context, w http.ResponseWriter, r *http.Request) (int, error) {
	var req QueryItemTokensReq
	if err := DecodeReq(r.Body, &req); err != nil {
		return http.StatusBadRequest, err
	}

	if err := validateQueryItemTokensReq(&req); err != nil {
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
	defer client.Close()

	var resp QueryItemTokensResp
	for i := 0; i < len(itemNames); i++ {
		resp = append(resp, &ItemTokenInfo{
			Name:   itemNames[i],
			Tokens: itemTokens[i],
		})
	}
	if err := EncodeResp(w, &resp); err != nil {
		return http.StatusInternalServerError, err
	}
	return http.StatusOK, nil
}

func validateQueryItemTokensReq(req *QueryItemTokensReq) error {
	if req.UserID == "" {
		return fmt.Errorf("missing user id")
	}
	return nil
}

// ******************************************
// ** END QueryItemTokens
// ******************************************

// ******************************************
// ** Begin QueryItems
// ******************************************

type QueryItemsReq struct {
	UserID   string `json:"user_id"`
	ItemName string `json:"item_name"`
}

type QueryItemsResp []*ItemInfo

type ItemInfo struct {
	DaysAgo   int     `json:"daysAgo"`
	HoursAgo  int     `json:"hoursAgo"`
	StoreName string  `json:"storeName"`
	StoreAddr string  `json:"storeAddress"`
	StoreLat  float64 `json:"storeLat"`
	StoreLng  float64 `json:"storeLong"`
	InStock   bool    `json:"inStock"`
	SeenCnt   int     `json:"seenCount"`
}

// QueryItems fetches the list of items in storage.
func QueryItems(ctx context.Context, w http.ResponseWriter, r *http.Request) (int, error) {
	var req QueryItemsReq
	if err := DecodeReq(r.Body, &req); err != nil {
		return http.StatusBadRequest, err
	}

	if err := cleanAndValidateQueryItemsReq(&req); err != nil {
		return http.StatusBadRequest, err
	}

	u, ok, err := GetUserInStorage(ctx, req.UserID)
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
	defer client.Close()

	resp := make(QueryItemsResp, 0)
	q := datastore.NewQuery(ItemKind).Filter("name =", req.ItemName)
	it := client.Run(ctx, q)
	for {
		var t Item
		_, err := it.Next(&t)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return http.StatusInternalServerError, fmt.Errorf("failed to query items: %v", err)
		}
		for _, itemInfo := range parseItem(&t) {
			resp = append(resp, itemInfo)
		}
	}

	if err := sortItems(resp, u.ZipCode); err != nil {
		return http.StatusInternalServerError, err
	}

	if err := EncodeResp(w, &resp); err != nil {
		return http.StatusInternalServerError, err
	}
	return http.StatusOK, nil
}

func cleanAndValidateQueryItemsReq(req *QueryItemsReq) error {
	req.ItemName = strings.ToLower(req.ItemName)
	if req.UserID == "" {
		return fmt.Errorf("missing user id")
	}
	if req.ItemName == "" {
		return fmt.Errorf("missing item name")
	}
	return nil
}

// ******************************************
// ** END QueryItems
// ******************************************

func parseItem(item *Item) []*ItemInfo {
	var res []*ItemInfo
	for _, stockReport := range item.StockReports {
		secondsAgo := int(time.Now().Unix() - stockReport.TimestampSec)
		itemInfo := &ItemInfo{
			DaysAgo:   secondsAgo / secondsToDay,
			HoursAgo:  secondsAgo / secondsToHour,
			StoreName: stockReport.StoreInfo.Name,
			StoreAddr: stockReport.StoreInfo.Addr,
			StoreLat:  stockReport.StoreInfo.Lat,
			StoreLng:  stockReport.StoreInfo.Long,
			InStock:   stockReport.InStock,
			SeenCnt:   stockReport.SeenCnt,
		}
		res = append(res, itemInfo)
	}
	return res
}

// Sort ItemInfo array by following priority.
// 1. Closest distance from store to user zip code.
// 2. Recent timestamp (time when item was seen at store)
func sortItems(resp QueryItemsResp, zipCode string) error {
	coords := zipCodeToLatLong[zipCode]
	lat := coords.Lat
	lng := coords.Long
	sort.Slice(resp, func(i, j int) bool {
		d1 := Distance(resp[i].StoreLat, resp[i].StoreLng, lat, lng)
		d2 := Distance(resp[j].StoreLat, resp[j].StoreLng, lat, lng)
		if d1 == d2 {
			return resp[i].HoursAgo < resp[j].HoursAgo
		}
		return d1 < d2
	})
	return nil
}
