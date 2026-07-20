package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync/atomic"
)

type apiConfig struct {
	fileserverHits atomic.Int32
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

type validResponse struct {
	Valid bool `json:"valid"`
}

type errorResponse struct {
	Error string `json:"error"`
}

type parameters struct {
	Body string `json:"body"`
}

func main() {
	mux := http.NewServeMux()

	cfg := &apiConfig{}
	mux.Handle("/app/", cfg.middlewareMetricsInc(http.StripPrefix("/app/", http.FileServer(http.Dir(".")))))

	mux.HandleFunc("GET /api/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(200)
		w.Write([]byte("OK"))
	})

	mux.HandleFunc("GET /admin/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(200)
		w.Write([]byte(fmt.Sprintf("<html>\n<body>\n<h1>Welcome, Chirpy Admin</h1>\n<p>Chirpy has been visited %d times!</p>\n</body>\n</html>", cfg.fileserverHits.Load())))
	})
	mux.HandleFunc("POST /admin/reset", func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Store(0)
	})

	mux.HandleFunc("POST /api/validate_chirp", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		decoder := json.NewDecoder(r.Body)
		params := parameters{}
		err := decoder.Decode(&params)
		if err != nil {
			log.Printf("Error decoding request body: %v", err)
			w.WriteHeader(500)
			return
		}
		if len(params.Body) > 140 {
			respondWithError(w, 400, "Chirp is too long") //Fix the error message to be JSON
		} else {
			respondWithJSON(w, 200, profanityFilter(params.Body))
		}

	})
	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}
	server.ListenAndServe()
}

func respondWithError(w http.ResponseWriter, code int, msg string) {
	log.Printf("Error %d: %s", code, msg)
	w.WriteHeader(code)
	w.Write([]byte(errorResponse{Error: msg}.Error)) //get these right

}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	resp, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshalling response: %v", err)
		w.WriteHeader(500)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)

	w.Write([]byte(resp))
}

func profanityFilter(body string) string {
	lower := strings.ToLower(body)
	fmt.Println(lower)
	words := strings.Split(lower, " ")
	fmt.Println(words)
	badwords := []string{"kerfuffle", "sharbert", "fornax"}
	//temp := body
	for _, word := range words {
		for _, badword := range badwords {
			if strings.ToLower(word) == badword {
				body = strings.Replace(body, word, "****", -1)
			} // Fix to replace any occurrence of bad word
		}
	}
	return body
}
