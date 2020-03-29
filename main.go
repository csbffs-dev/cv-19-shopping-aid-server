package main

import (
	"context"
	"log"
	"net/http"
	"os"
)

func main() {
	// TODO: Use github.com/gorilla/mux for HTTP routing.
	http.HandleFunc("/user/setup", userSetupHandler)
	http.HandleFunc("/item/query", itemQueryHandler)
	http.HandleFunc("/store/query", storeQueryHandler)
	http.HandleFunc("/report/upload", reportUploadHandler)
	http.HandleFunc("/receipt/parse", receiptParseHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Printf("Defaulting to port %s", port)
	}

	log.Printf("Listening on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

func userSetupHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	if r.Method != "POST" {
		http.NotFound(w, r)
		return
	}
	if status, err := SetupUser(ctx, r); err != nil {
		http.Error(w, err.Error(), status)
	}
}

func itemQueryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.NotFound(w, r)
		return
	}
	status, err := QueryItems(r)
	if err != nil {
		http.Error(w, err.Error(), status)
	}
}

func storeQueryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.NotFound(w, r)
		return
	}
	status, err := QueryStores(r)
	if err != nil {
		http.Error(w, err.Error(), status)
	}
}

func reportUploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.NotFound(w, r)
		return
	}
	status, err := UploadReport(r)
	if err != nil {
		http.Error(w, err.Error(), status)
	}
}
func receiptParseHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.NotFound(w, r)
		return
	}
	status, err := ParseReceipt(r)
	if err != nil {
		http.Error(w, err.Error(), status)
	}
}
