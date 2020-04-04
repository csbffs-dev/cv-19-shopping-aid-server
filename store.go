package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"

	"cloud.google.com/go/datastore"
	"github.com/google/uuid"
	"google.golang.org/api/iterator"
	"googlemaps.github.io/maps"
)

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
	StoreID string `datastore:"storeID" json:"store_id"`
	Name    string `datastore:"name" json:"name"`
	Addr    string `datastore:"addr" json:"address"`
}

// ******************************************
// ** BEGIN QueryStores
// ******************************************

type QueryStoresReq struct {
	UserID string `json:"user_id"`
}

type QueryStoresResp struct {
	Stores []*Store `json:"stores"`
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

	resp := &QueryStoresResp{Stores: make([]*Store, 0)}
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
		resp.Stores = append(resp.Stores, &st)
	}

	if err := sortAndPruneNearby(resp, u.ZipCode); err != nil {
		return http.StatusInternalServerError, err
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

// vetStoreInfo vets the storeInfo name and address before adding it to Storage.
// 1. calls the Google Maps Places API with a query with storeInfo name and address.
// 2. places API returns the fully qualified name and address of the candidate place that matches.
//    Only one candidate place can be returned, otherwise an error is returned with string output of the candidate places.
// 3. checks to see that the storeInfo name exists as substring in the fully qualified name. Similarly, for the addresses.
//    If they don't match, an error is returned with string output of both.
//    If they do match, the storeInfo is updated with the fully qualified data.
func vetStoreInfo(ctx context.Context, client *maps.Client, storeInfo *Store) error {
	placesQueryInput := fmt.Sprintf("%s %s", storeInfo.Name, storeInfo.Addr)

	findPlaceReq := &maps.FindPlaceFromTextRequest{
		InputType: maps.FindPlaceFromTextInputTypeTextQuery,
		Input:     placesQueryInput,
		Fields: []maps.PlaceSearchFieldMask{
			maps.PlaceSearchFieldMaskFormattedAddress,
			maps.PlaceSearchFieldMaskName,
			maps.PlaceSearchFieldMaskPlaceID,
		},
	}
	findPlaceResp, err := client.FindPlaceFromText(ctx, findPlaceReq)
	if err != nil {
		return err
	}

	if len(findPlaceResp.Candidates) != 1 {
		log.Printf("the store info `%s` returned %d matches", storeInfo, len(findPlaceResp.Candidates))
		errMsg := fmt.Sprintf("found %d store(s) that matched the given store information, but only 1 store can match.\n", len(findPlaceResp.Candidates))
		for i, cand := range findPlaceResp.Candidates {
			errMsg += fmt.Sprintf("%d: %s %s\n", i+1, cand.Name, cand.FormattedAddress)
		}
		return fmt.Errorf(errMsg)
	}

	vettedName := findPlaceResp.Candidates[0].Name
	vettedAddr := strings.TrimSuffix(findPlaceResp.Candidates[0].FormattedAddress, ", United States")

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

	log.Printf("store `%q %q` vetted and changed to `%q %q`", storeInfo.Name, storeInfo.Addr, vettedName, vettedAddr)
	storeInfo.Name = vettedName
	storeInfo.Addr = vettedAddr
	return nil
}

func sortAndPruneNearby(resp *QueryStoresResp, zipCode string) error {
	// TODO: Order resp.Stores based on closest distance between user zip code and store address.
	// 1. Compare zip codes between store and user. Filter stores that are in different state/region as user.
	// 2. Pass in the store addresses and zipcode to Google Maps Distance Matrix API.
	// 3. Order stores based on response.
	// 4. Return the top 10.
	return nil
}
