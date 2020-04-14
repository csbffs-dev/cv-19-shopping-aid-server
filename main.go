package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

func main() {
	r := mux.NewRouter()
	// TODO: Set up admin endpoints.
	r.HandleFunc("/user/setup", userSetupHandler)
	r.HandleFunc("/user/edit", userEditHandler)
	r.HandleFunc("/user/delete", userDeleteHandler)
	r.HandleFunc("/user/query", userQueryHandler)
	r.HandleFunc("/item/query", itemQueryHandler)
	r.HandleFunc("/item/tokens/query", itemTokensQueryHandler)
	r.HandleFunc("/store/query", storeQueryHandler)
	r.HandleFunc("/store/add", storeAddHandler)
	r.HandleFunc("/report/upload", reportUploadHandler)
	r.HandleFunc("/receipt/parse", receiptParseHandler)
	hr := cors.Default().Handler(r)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Printf("Defaulting to port %s", port)
	}

	log.Printf("Listening on port %s", port)
	if err := http.ListenAndServe(":"+port, hr); err != nil {
		log.Fatal(err)
	}
}

func userSetupHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	if r.Method != "POST" {
		http.NotFound(w, r)
		return
	}
	if status, err := SetupUser(ctx, w, r); err != nil {
		http.Error(w, err.Error(), status)
	}
}

func userEditHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	if r.Method != "POST" {
		http.NotFound(w, r)
		return
	}
	if status, err := EditUser(ctx, w, r); err != nil {
		http.Error(w, err.Error(), status)
	}
}

func userDeleteHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	if r.Method != "POST" {
		http.NotFound(w, r)
		return
	}
	if status, err := DeleteUser(ctx, w, r); err != nil {
		http.Error(w, err.Error(), status)
	}
}

func userQueryHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	if r.Method != "POST" {
		http.NotFound(w, r)
		return
	}
	if status, err := QueryUser(ctx, w, r); err != nil {
		http.Error(w, err.Error(), status)
	}
}

func itemQueryHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	if r.Method != "POST" {
		http.NotFound(w, r)
		return
	}
	status, err := QueryItems(ctx, w, r)
	if err != nil {
		http.Error(w, err.Error(), status)
	}
}

func itemTokensQueryHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	if r.Method != "POST" {
		http.NotFound(w, r)
		return
	}
	status, err := QueryItemTokens(ctx, w, r)
	if err != nil {
		http.Error(w, err.Error(), status)
	}
}

func storeQueryHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	if r.Method != "POST" {
		http.NotFound(w, r)
		return
	}
	status, err := QueryStores(ctx, w, r)
	if err != nil {
		http.Error(w, err.Error(), status)
	}
}

func storeAddHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	if r.Method != "POST" {
		http.NotFound(w, r)
		return
	}
	status, err := AddStore(ctx, w, r)
	if err != nil {
		http.Error(w, err.Error(), status)
	}
}

func reportUploadHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	if r.Method != "POST" {
		http.NotFound(w, r)
		return
	}
	status, err := UploadReport(ctx, w, r)
	if err != nil {
		http.Error(w, err.Error(), status)
	}
}

func receiptParseHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	if r.Method != "POST" {
		http.NotFound(w, r)
		return
	}
	status, err := ParseReceipt(ctx, w, r)
	if err != nil {
		http.Error(w, err.Error(), status)
	}
}
