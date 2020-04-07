package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"sort"
	"strings"

	"cloud.google.com/go/datastore"
	"github.com/google/uuid"
	"google.golang.org/api/iterator"
	"googlemaps.github.io/maps"
)

var (
	validAddress *regexp.Regexp
)

func init() {
	validAddress = regexp.MustCompile("^.+, .+, [A-Za-z]{2,} [0-9]{5,}$")
}

// See https://developers.google.com/places/web-service/supported_types#table1 for all place types.
var (
	relevantStoreTypes = map[string]bool{
		"convenience_store":      true,
		"department_store":       true,
		"drugstore":              true,
		"grocery_or_supermarket": true,
		"liquor_store":           true,
		"pharmacy":               true,
		"supermarket":            true,
	}
)

type Store struct {
	StoreID string  `datastore:"storeID" json:"store_id"`
	Name    string  `datastore:"name" json:"name"`
	Addr    string  `datastore:"addr" json:"address"`
	Lat     float64 `datastore:"lat" json:"latitude"`
	Long    float64 `datastore:"long" json:"longitude"`
}

// ******************************************
// ** BEGIN QueryStores
// ******************************************

const (
	// Maximum number of stores to return in QueryStores
	queryStoresLimit = 10
)

type QueryStoresReq struct {
	UserID string `json:"user_id"`
}

type QueryStoresResp struct {
	Stores []*QueryStoreInfo `json:"stores"`
}

type QueryStoreInfo struct {
	*Store
	*Address
}

type Address struct {
	Street  string `json:"street"`
	City    string `json:"city"`
	State   string `json:"state"`
	ZipCode string `json:"zip_code"`
}

// QueryStores fetches the list of stores in storage.
func QueryStores(ctx context.Context, w http.ResponseWriter, r *http.Request) (int, error) {
	var req QueryStoresReq
	if err := DecodeReq(r.Body, &req); err != nil {
		return http.StatusBadRequest, err
	}

	if err := validateQueryStoresReq(req); err != nil {
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

	var stores []*Store
	q := datastore.NewQuery(StoreKind)
	it := client.Run(ctx, q)
	for {
		var st Store
		_, err := it.Next(&st)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return http.StatusInternalServerError, fmt.Errorf("failed to query for all stores: %v", err)
		}
		stores = append(stores, &st)
	}

	// TODO: Use a heap instead of sort function to optimize getting the top
	// `queryStoresLimit` stores from the stores list.
	if err := sortStoresByDistance(stores, u.ZipCode); err != nil {
		return http.StatusInternalServerError, err
	}
	if len(stores) > queryStoresLimit {
		stores = stores[:queryStoresLimit]
	}

	resp := &QueryStoresResp{}
	for _, st := range stores {
		addr, err := parseAddressComponents(st.Addr)
		if err != nil {
			log.Fatalf("failed to parse address %q: %v", st.Addr, err)
			continue
		}
		resp.Stores = append(resp.Stores, &QueryStoreInfo{Store: st, Address: addr})
	}

	if err := EncodeResp(w, &resp); err != nil {
		return http.StatusInternalServerError, err
	}
	return http.StatusOK, nil
}

func validateQueryStoresReq(req QueryStoresReq) error {
	if req.UserID == "" {
		return fmt.Errorf("missing user id")
	}
	return nil
}

func parseAddressComponents(address string) (*Address, error) {
	if !validAddress.MatchString(address) {
		return nil, fmt.Errorf("address does not follow standard format `<street>, <city>, <state> <zip code>`")
	}
	components := strings.Split(address, ", ")
	stateAndZipCode := strings.Split(components[2], " ")
	return &Address{
		Street:  strings.TrimSpace(components[0]),
		City:    strings.TrimSpace(components[1]),
		State:   strings.TrimSpace(stateAndZipCode[0]),
		ZipCode: strings.TrimSpace(stateAndZipCode[1]),
	}, nil
}

// ******************************************
// ** END QueryStores
// ******************************************

// ******************************************
// ** BEGIN AddStore
// ******************************************

type AddStoreReq struct {
	UserID   string `json:"user_id"`
	Name     string `json:"name"`
	AddrText string `json:"address"`
}

type AddStoreResp struct {
	StoreID string `json:"store_id"`
}

func AddStore(ctx context.Context, w http.ResponseWriter, r *http.Request) (int, error) {
	var req AddStoreReq
	if err := DecodeReq(r.Body, &req); err != nil {
		return http.StatusBadRequest, err
	}

	if err := cleanAndValidateAddStoreReq(&req); err != nil {
		return http.StatusBadRequest, err
	}

	_, ok, err := GetUserInStorage(ctx, req.UserID)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to check user creds: %v", err)
	}
	if !ok {
		return http.StatusForbidden, fmt.Errorf("user id is invalid: %q", req.UserID)
	}

	st := &Store{
		Name: req.Name,
		Addr: req.AddrText,
	}

	// TODO: Prevent dupes. Check that store does not already exist in storage.

	client, err := MapsClient()
	if err != nil {
		return http.StatusInternalServerError, err
	}

	if err := vetStoreInfo(ctx, client, st); err != nil {
		return http.StatusBadRequest, err
	}

	uid, err := uuid.NewRandom()
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to generate store id: %v", err)
	}
	st.StoreID = uid.String()

	if err := createStoreInStorage(ctx, st); err != nil {
		return http.StatusInternalServerError, err
	}

	resp := &AddStoreResp{
		StoreID: st.StoreID,
	}
	if err := EncodeResp(w, &resp); err != nil {
		return http.StatusInternalServerError, err
	}
	return http.StatusOK, nil
}

func cleanAndValidateAddStoreReq(req *AddStoreReq) error {
	req.Name = strings.TrimSpace(req.Name)
	req.AddrText = strings.TrimSpace(req.AddrText)

	if req.UserID == "" {
		return fmt.Errorf("missing user id")
	}
	if req.Name == "" {
		return fmt.Errorf("missing store name")
	}
	if req.AddrText == "" {
		return fmt.Errorf("missing store address text")
	}
	return nil
}

// ******************************************
// ** END AddStore
// ******************************************

// GetStoreInStorage fetches the store with key = storeID in storage.
// Returns a non-nil error if storage client experienced a failure.
func GetStoreInStorage(ctx context.Context, storeID string) (*Store, error) {
	client, err := StorageClient(ctx)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	var st Store
	key := datastore.NameKey(StoreKind, storeID, nil)
	if err := client.Get(ctx, key, &st); err != nil {
		return nil, fmt.Errorf("failed to get store from storage: %v", err)
	}
	return &st, nil
}

func createStoreInStorage(ctx context.Context, st *Store) error {
	client, err := StorageClient(ctx)
	if err != nil {
		return err
	}
	defer client.Close()

	key := datastore.NameKey(StoreKind, st.StoreID, nil)
	_, err = client.Put(ctx, key, st)
	if err != nil {
		return fmt.Errorf("failed to add store in storage: %v", err)
	}
	return nil
}

// vetStoreInfo vets the storeInfo before adding it to Storage.
// 1. calls the Google Maps Places API with a query `<storeInfo.name> <storeInfo.address>`.
// 2. Places API returns the fully qualified name, address, lat, and long of the candidate
//    place that matches.
//    Only one candidate place can be returned, otherwise an error is returned with string
//    output of the candidate places.
// 3. calls the Places API again to get details of the candidate place. If the candidate
//    does not have a relevant label (see relevantStoreTypes variable), the candidate
//    is rejected and an error is returned.
// 4. overrides storeInfo fields with those returned by Places API
func vetStoreInfo(ctx context.Context, client *maps.Client, storeInfo *Store) error {
	placesQueryInput := fmt.Sprintf("%s %s", storeInfo.Name, storeInfo.Addr)

	findPlaceReq := &maps.FindPlaceFromTextRequest{
		InputType: maps.FindPlaceFromTextInputTypeTextQuery,
		Input:     placesQueryInput,
		Fields: []maps.PlaceSearchFieldMask{
			maps.PlaceSearchFieldMaskFormattedAddress,
			maps.PlaceSearchFieldMaskName,
			maps.PlaceSearchFieldMaskPlaceID,
			maps.PlaceSearchFieldMaskGeometry,
		},
	}
	findPlaceResp, err := client.FindPlaceFromText(ctx, findPlaceReq)
	if err != nil {
		return err
	}

	if len(findPlaceResp.Candidates) != 1 {
		log.Printf("the store info `%q %q` returned %d matches", storeInfo.Name, storeInfo.Addr, len(findPlaceResp.Candidates))
		errMsg := fmt.Sprintf("found %d store(s) that matched the given store information, but only 1 store can match.\n", len(findPlaceResp.Candidates))
		for i, cand := range findPlaceResp.Candidates {
			errMsg += fmt.Sprintf("%d: %s %s\n", i+1, cand.Name, cand.FormattedAddress)
		}
		return fmt.Errorf(errMsg)
	}

	vettedName := findPlaceResp.Candidates[0].Name
	vettedAddr := strings.TrimSuffix(findPlaceResp.Candidates[0].FormattedAddress, ", United States")
	lat := findPlaceResp.Candidates[0].Geometry.Location.Lat
	lng := findPlaceResp.Candidates[0].Geometry.Location.Lng

	detailsReq := &maps.PlaceDetailsRequest{
		PlaceID: findPlaceResp.Candidates[0].PlaceID,
		Fields:  []maps.PlaceDetailsFieldMask{maps.PlaceDetailsFieldMaskTypes},
	}
	detailsResp, err := client.PlaceDetails(ctx, detailsReq)
	for _, placeType := range detailsResp.Types {
		if _, ok := relevantStoreTypes[placeType]; ok {
			break
		}
		return fmt.Errorf("could not verify store info `%q %q` as a real grocery store", vettedName, vettedAddr)
	}

	log.Printf("store `%q %q` vetted and changed to `%q %q (%f, %f)`", storeInfo.Name, storeInfo.Addr, vettedName, vettedAddr, lat, lng)
	storeInfo.Name = vettedName
	storeInfo.Addr = vettedAddr
	storeInfo.Lat = lat
	storeInfo.Long = lng
	return nil
}

func sortStoresByDistance(stores []*Store, zipCode string) error {
	coords := zipCodeToLatLong[zipCode]
	lat := coords.Lat
	lng := coords.Long
	sort.Slice(stores, func(i, j int) bool {
		return Distance(stores[i].Lat, stores[i].Long, lat, lng) <
			Distance(stores[j].Lat, stores[j].Long, lat, lng)
	})
	if len(stores) > queryStoresLimit {
		stores = stores[:queryStoresLimit]
	}
	return nil
}
