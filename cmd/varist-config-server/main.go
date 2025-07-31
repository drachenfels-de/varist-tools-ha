package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"

	"github.com/gorilla/mux"
)

var (
	filePath string
	port     string
	mutex    sync.Mutex
)

type Indicator struct {
	Description string `json:"Description"`
	Action      string `json:"Action"`
	MaxCount    int    `json:"MaxCount"`
	Regex       string `json:"Regex,omitempty"`
}

type Root struct {
	IndicatorCategories []map[string]Indicator `json:"Indicator Categories"`
	IndicatorItems      []map[string]Indicator `json:"IndicatorItems"`
}

func loadJSON() (*Root, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	var root Root
	err = json.Unmarshal(data, &root)
	return &root, err
}

func saveJSON(root *Root) error {
	data, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, data, 0o644)
}

func getDataHandler(w http.ResponseWriter, r *http.Request) {
	mutex.Lock()
	defer mutex.Unlock()

	root, err := loadJSON()
	if err != nil {
		http.Error(w, "Cannot load JSON: "+err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(root)
}

func saveDataHandler(w http.ResponseWriter, r *http.Request) {
	mutex.Lock()
	defer mutex.Unlock()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}

	var root Root
	if err := json.Unmarshal(body, &root); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if err := saveJSON(&root); err != nil {
		http.Error(w, "Save failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write([]byte("Saved successfully"))
}

func main() {
	flag.StringVar(&filePath, "file", "hybrid-analyzer.json", "Path to JSON file")
	flag.StringVar(&port, "port", "8100", "Listen port")
	flag.Parse()

	r := mux.NewRouter()
	r.HandleFunc("/api/data", getDataHandler).Methods("GET")
	r.HandleFunc("/api/save", saveDataHandler).Methods("POST")
	r.PathPrefix("/").Handler(http.FileServer(http.Dir("./static/")))

	fmt.Printf("Server listening at http://localhost:%s\n", port)
	http.ListenAndServe(":"+port, r)
}
