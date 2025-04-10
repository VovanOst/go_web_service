package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// jsonResponse выводит данные в формате JSON.
func jsonResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		fmt.Printf("error encoding response: %v", err)
	}
}

func writeJSON(w http.ResponseWriter, data interface{}, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		fmt.Printf("error encoding JSON response: %v", err)
	}
}
