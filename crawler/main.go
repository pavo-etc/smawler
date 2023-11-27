package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

type Request struct {
	URL string `json:"url"`
}

var VERBOSE bool = false
var TIMEOUT int = 10000

func main() {

	var cli bool
	flag.BoolVar(&cli, "cli", false, "whether to use cli interface")

	var resetDb bool
	flag.BoolVar(&resetDb, "resetDb", false, "whether to reset the database")

	var verbose bool
	flag.BoolVar(&verbose, "verbose", false, "whether to enable verbose logging")

	var timeout int
	flag.IntVar(&timeout, "timeout", 10000, "timeout on db locking")

	flag.Parse()

	VERBOSE = verbose
	TIMEOUT = timeout

	if resetDb {
		ResetDatabase()
	}

	if cli {
		startCLI()
		return
	}

	http.Handle("/", http.FileServer(http.Dir("./static")))
	http.HandleFunc("/reset", resetHandler)
	http.HandleFunc("/reject", rejectHandler)
	http.HandleFunc("/approve", approveHandler)
	http.HandleFunc("/getNew", getNewHandler)

	err := http.ListenAndServe(":3000", nil)
	if errors.Is(err, http.ErrServerClosed) {
		fmt.Printf("server closed\n")
	} else if err != nil {
		fmt.Printf("error starting server: %s\n", err)
		os.Exit(1)
	}
}

func resetHandler(w http.ResponseWriter, r *http.Request) {
	ResetDatabase()
}

func rejectHandler(w http.ResponseWriter, r *http.Request) {
	// get string domain from request body
	decoder := json.NewDecoder(r.Body)
	var req Request
	err := decoder.Decode(&req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	url := req.URL
	fmt.Println("req:", req)
	RejectDomain(url)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func approveHandler(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var req Request
	err := decoder.Decode(&req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	url := req.URL
	go CrawlDomain(url)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func getNewHandler(w http.ResponseWriter, r *http.Request) {
	domains := GetNewDomains()
	// domains is a slice of strings
	// send res {domains: domains}
	res := map[string][]string{"domains": domains}
	json.NewEncoder(w).Encode(res)
}
