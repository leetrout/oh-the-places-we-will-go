package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"math/rand"
	"net/http"
	"os"
	"strconv"
)

// We don't want to handle anything JS can't use
// https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Number/MAX_SAFE_INTEGER
const jsMax = 9007199254740991

var jsMaxInt = big.NewInt(jsMax)

// mimeAppJSON is the content type header for all our responses
var mimeAppJSON = []string{"application/json"}

// AuthResp is returned as JSON from /auth
type AuthResp struct {
	Token int64 `json:"token,string"`
}

// ProductResp is returned as JSON from /product
type ProductResp struct {
	Result string `json:"result"`
}

// handleAuth returns AuthResp containing a random int64.
// Only supports GET requests.
func handleAuth(w http.ResponseWriter, r *http.Request) {
	headers := w.Header()
	headers["Content-Type"] = mimeAppJSON

	// Ensure we have a GET request
	if r.Method != http.MethodGet {
		http.Error(w, fmt.Sprintf("{\"detail\": \"%s is not supported\"}", r.Method), http.StatusBadRequest)
		return
	}

	// Generate a large pseudo-random number and return it or a 500 error
	authResp := &AuthResp{rand.Int63()}
	body, err := json.Marshal(authResp)
	if err != nil {
		http.Error(w, fmt.Sprintf("{\"detail\": \"%s\"}", err.Error()), http.StatusInternalServerError)
		return
	}
	_, err = w.Write(body)
	if err != nil {
		log.Println("could not write body", err)
	}
}

// handleProduct calculates the product for two integers, a & b.
// TODO Could reduce complexity with a function to handle checks
func handleProduct(w http.ResponseWriter, r *http.Request) {
	headers := w.Header()
	headers["Content-Type"] = mimeAppJSON

	// Ensure we have a GET request
	if r.Method != http.MethodGet {
		http.Error(w, fmt.Sprintf("{\"detail\": \"%s is not supported\"}", r.Method), http.StatusBadRequest)
		return
	}

	// Check for a valid int64 in the Authorization header
	auth, ok := r.Header["Authorization"]
	if !ok {
		http.Error(w, "{\"detail\": \"'Authorization' is a required header\"}", http.StatusBadRequest)
		return
	}

	_, err := strconv.ParseInt(auth[0], 10, 64)
	if err != nil {
		http.Error(w, "{\"detail\": \"invalid authorization token\"}", http.StatusBadRequest)
		return
	}

	// Pull out the params
	a, ok := r.URL.Query()["a"]
	if !ok {
		http.Error(w, "{\"detail\": \"'a' is a required parameter\"}", http.StatusBadRequest)
		return
	}

	b, ok := r.URL.Query()["b"]
	if !ok {
		http.Error(w, "{\"detail\": \"'b' is a required parameter\"}", http.StatusBadRequest)
		return
	}

	// Ensure params are only specified once
	if len(a) != 1 {
		http.Error(w, "{\"detail\": \"'a' must only be specified once\"}", http.StatusBadRequest)
		return
	}

	if len(b) != 1 {
		http.Error(w, "{\"detail\": \"'b' must only be specified once\"}", http.StatusBadRequest)
		return
	}

	// Ensure the params are integers
	aint, err := strconv.ParseInt(a[0], 10, 64)
	if err != nil {
		http.Error(w, fmt.Sprintf("{\"detail\": \"'a' must be an integer between -%d and %d\"}", jsMax, jsMax), http.StatusBadRequest)
		return
	}

	bint, err := strconv.ParseInt(b[0], 10, 64)
	if err != nil {
		http.Error(w, fmt.Sprintf("{\"detail\": \"'b' must be an integer between -%d and %d\"}", jsMax, jsMax), http.StatusBadRequest)
		return
	}

	// Perform math as big ints to avoid overflows.
	abig := big.NewInt(aint)
	bbig := big.NewInt(bint)
	result := big.NewInt(0)
	result.Mul(abig, bbig)

	// Ensure the value returned is OK for JS to consume
	if jsMaxInt.CmpAbs(result) == -1 {
		http.Error(w, "{\"detail\": \"result is larger than Javascript's Number.MAX_SAFE_INTEGER.\"}", http.StatusBadRequest)
		return
	}

	// Return product info
	productResp := &ProductResp{result.String()}
	body, err := json.Marshal(productResp)
	if err != nil {
		http.Error(w, fmt.Sprintf("{\"detail\": \"%s\"}", err.Error()), http.StatusInternalServerError)
		return
	}
	_, err = w.Write(body)
	if err != nil {
		log.Println("could not write body", err)
	}
}

// handleAny presents the index page for any unknown paths
func handleAny(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "index.html")
}

func main() {
	// Route handlers
	http.HandleFunc("/", handleAny)
	http.HandleFunc("/auth", handleAuth)
	http.HandleFunc("/product", handleProduct)

	// Preflight
	log.Println("Preflight checks...")
	_, err := os.Stat("./index.html")
	if os.IsNotExist(err) {
		log.Fatal("Could not locate ./index.html")
	}

	// Serve
	log.Println("Listening for requests on port 9000")
	log.Fatal(http.ListenAndServe(":9000", nil))
}
