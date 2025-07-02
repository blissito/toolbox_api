package main

import (
	"encoding/json"
	"log"
	"net/http"
	"toolbox/tools"
)

type ToolRequest struct {
	Name    string                 `json:"tool"`
	Payload map[string]interface{} `json:"payload"`
}

type ToolResponse struct {
	Success bool        `json:"success"`
	Result  interface{} `json:"result,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func toolHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ToolRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, ToolResponse{
			Success: false,
			Error:   "Invalid request",
		})
		return
	}

	var result interface{}
	var err error

	switch req.Name {
	case "webfetch":
		result, err = tools.WebFetch(req.Payload)
	default:
		respondJSON(w, http.StatusBadRequest, ToolResponse{
			Success: false,
			Error:   "Tool not found",
		})
		return
	}

	if err != nil {
		respondJSON(w, http.StatusInternalServerError, ToolResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	respondJSON(w, http.StatusOK, ToolResponse{
		Success: true,
		Result:  result,
	})
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, "index.html")
}

func docsHandler(w http.ResponseWriter, r *http.Request) {
	// Servir el archivo de documentación
	if r.URL.Path == "/docs/webfetch" || r.URL.Path == "/docs/webfetch/" {
		http.ServeFile(w, r, "docs/webfetch.html")
		return
	}
	// Redirigir si se accede a /docs sin especificar el archivo
	http.Redirect(w, r, "/docs/webfetch", http.StatusFound)
}

func main() {
	// Servir archivos estáticos
	fs := http.FileServer(http.Dir("."))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// Rutas de la API
	http.HandleFunc("/api/tool", toolHandler)
	
	// Ruta de documentación
	http.HandleFunc("/docs/webfetch", docsHandler)
	http.HandleFunc("/docs/webfetch/", docsHandler)
	
	// Redirigir /docs a /docs/webfetch
	http.HandleFunc("/docs", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/docs/webfetch", http.StatusMovedPermanently)
	})
	
	// Ruta de inicio
	http.HandleFunc("/", homeHandler)

	port := ":8000"
	log.Printf("Server starting on %s...\n", port)
	log.Fatal(http.ListenAndServe(port, nil))
}
