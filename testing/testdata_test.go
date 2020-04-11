// Prepopulates data in local datastore emulator.
//
// Usage:
// 1. Make sure you have the server, datastore emulator, and datastore gui up and running.
// 2. Run the following commands.
//    $ cd path/to/cv19-shopping-aid-server/testing
//    $ go test -v
//
// Each go unit test represents the following workflow.
// 1. Setup the users
// 2. Add the stores.
// 3. Report items.
//
// These unit tests run in parallel.
// --> WARNING: Each unit test is standalone! One unit test should not depend on data (i.e. users, stores)
//     from another unit test.
//
// To add your data, create another unit test at the end of file like so.
//
// func TestDo<NEXT_DIGIT>(t *testing.T) {
//     ... // See TestDo1 as an example
// }
//
// Then, instead of `go test -v`, do `go test -v -run=TestDo<NEXT_DIGIT>`
package testdata

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"testing"
)

const (
	devHostAddr          = "http://localhost:8080"
	userSetupEndpoint    = "/user/setup"
	storeAddEndpoint     = "/store/add"
	reportUploadEndpoint = "/report/upload"
)

var client *http.Client

type SetupUserReq struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	ZipCode   string `json:"zip_code"`
}

type SetupUserResp struct {
	UserID string `json:"user_id"`
}

type AddStoreReq struct {
	UserID   string `json:"user_id"`
	Name     string `json:"name"`
	AddrText string `json:"address"`
}

type AddStoreResp struct {
	StoreID string `json:"store_id"`
}

type UploadReportReq struct {
	UserID   string   `json:"user_id"`
	StoreID  string   `json:"store_id"`
	InStock  []string `json:"in_stock_items"`
	OutStock []string `json:"out_stock_items"`
}

func TestMain(m *testing.M) {
	client = &http.Client{}
	os.Exit(m.Run())
}

func TestDo1(t *testing.T) {
	t.Parallel()

	// Setup the users.
	setupUserReqs := []*SetupUserReq{
		{
			FirstName: "Tony",
			LastName:  "Stark",
			ZipCode:   "98109",
		},
		{
			FirstName: "Peter",
			LastName:  "Parker",
			ZipCode:   "98101",
		},
	}
	var setupUserResps []*SetupUserResp

	for _, r := range setupUserReqs {
		ur, err := setupUser(client, r)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("Created User %v", ur.UserID)
		setupUserResps = append(setupUserResps, ur)
	}

	// Add the stores.
	// Tony Stark is adding both stores.
	addStoreReqs := []*AddStoreReq{
		{
			UserID:   setupUserResps[0].UserID,
			Name:     "Costco",
			AddrText: "Kirkland",
		},
		{
			UserID:   setupUserResps[0].UserID,
			Name:     "Costco",
			AddrText: "Seattle",
		},
	}
	var addStoreResps []*AddStoreResp

	for _, r := range addStoreReqs {
		sr, err := addStore(client, r)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("Created Store %v", sr.StoreID)
		addStoreResps = append(addStoreResps, sr)
	}

	// Report items.
	// Tony Stark adds the first report to Costco Kirkland.
	// Tony Stark also adds the second report but to Costco Seattle.
	// Peter Parker adds the third report to Costco Seattle.
	uploadReportReqs := []*UploadReportReq{
		{
			UserID:   setupUserResps[0].UserID,
			StoreID:  addStoreResps[0].StoreID,
			InStock:  []string{"chicken breast", "toilet paper"},
			OutStock: []string{"hand sanitizer"},
		},
		{
			UserID:   setupUserResps[0].UserID,
			StoreID:  addStoreResps[1].StoreID,
			InStock:  []string{"hand sanitizer"},
			OutStock: []string{"toilet paper", "paper towels"},
		},
		{
			UserID:  setupUserResps[1].UserID,
			StoreID: addStoreResps[1].StoreID,
			InStock: []string{"hand sanitizer"},
		},
	}

	for _, r := range uploadReportReqs {
		if err := uploadReport(client, r); err != nil {
			t.Fatal(err)
		}
		t.Log("Uploaded report")
	}
}

func TestDo2(t *testing.T) {
	t.Parallel()

	ur, err := setupUser(client, &SetupUserReq{"Steve", "Rogers", "98083"})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Created User %v", ur.UserID)

	sr, err := addStore(client, &AddStoreReq{UserID: ur.UserID, Name: "Trader Joe's", AddrText: "Capitol Hill"})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Created Store %v", sr.StoreID)

	if err := uploadReport(client, &UploadReportReq{
		UserID:  ur.UserID,
		StoreID: sr.StoreID,
		InStock: []string{"cheddar", "chicken breast", "flour"},
	}); err != nil {
		t.Fatal(err)
	}
	t.Log("Uploaded report")
}

func TestDo3(t *testing.T) {
	t.Parallel()

	ur, err := setupUser(client, &SetupUserReq{"Bruce", "Banner", "98402"})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Created User %v", ur.UserID)

	sr1, err := addStore(client, &AddStoreReq{UserID: ur.UserID, Name: "Uwajimaya", AddrText: "Seattle"})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Created Store %v", sr1.StoreID)

	sr2, err := addStore(client, &AddStoreReq{UserID: ur.UserID, Name: "H Mart", AddrText: "Pike Place"})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Created Store %v", sr2.StoreID)

	if err := uploadReport(client, &UploadReportReq{
		UserID:   ur.UserID,
		StoreID:  sr1.StoreID,
		InStock:  []string{"brown rice", "chicken breast"},
		OutStock: []string{"flour"},
	}); err != nil {
		t.Fatal(err)
	}
	t.Log("Uploaded report")

	if err := uploadReport(client, &UploadReportReq{
		UserID:   ur.UserID,
		StoreID:  sr2.StoreID,
		OutStock: []string{"pasta"},
	}); err != nil {
		t.Fatal(err)
	}
	t.Log("Uploaded report")

	// Although this request should succeed, there should not be a
	// duplicate report under the item.
	if err := uploadReport(client, &UploadReportReq{
		UserID:   ur.UserID,
		StoreID:  sr2.StoreID,
		OutStock: []string{"pasta"},
	}); err != nil {
		t.Fatal(err)
	}
	t.Log("Uploaded report")
}

func setupUser(client *http.Client, req *SetupUserReq) (*SetupUserResp, error) {
	var resp SetupUserResp
	if err := doPost(userSetupEndpoint, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func addStore(client *http.Client, req *AddStoreReq) (*AddStoreResp, error) {
	var resp AddStoreResp
	if err := doPost(storeAddEndpoint, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func uploadReport(client *http.Client, req *UploadReportReq) error {
	if err := doPost(reportUploadEndpoint, req, nil); err != nil {
		return err
	}
	return nil
}

func doPost(endpoint string, reqData, respData interface{}) error {
	buf, err := json.Marshal(reqData)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %v", err)
	}
	req, err := http.NewRequest("POST", devHostAddr+endpoint, bytes.NewBuffer(buf))
	if err != nil {
		return fmt.Errorf("failed to set up request: %v", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()
	if respData == nil {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
		return fmt.Errorf("failed to decode response: %v", err)
	}
	return nil
}
