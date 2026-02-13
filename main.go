package main

import (
	_ "github.com/lib/pq"
	"github.com/joho/godotenv"
	"github.com/Tanner-Denti/chirpy/internal/database"
	"log"
	"net/http"
	"sync/atomic"
	"os"
	"database/sql"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	db *database.Queries
	platform string
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal(err)
	}

	const filepathRoot = "."
	const port = "8080"

	dbUrl := os.Getenv("DB_URL")
	db, err := sql.Open("postgres", dbUrl)
	if err != nil {
		log.Fatal(err)
	}
	dbQueries := database.New(db)

	platform := os.Getenv("PLATFORM")

	apiCfg := apiConfig { 
		fileserverHits: atomic.Int32{},
		db: dbQueries,
		platform: platform,
	}

	mux := http.NewServeMux()
	fsHandler := apiCfg.middlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir(filepathRoot))))
	mux.Handle("/app/", fsHandler)

	mux.HandleFunc("GET /api/healthz", handlerHealthzGet)
	mux.HandleFunc("POST /api/users", apiCfg.handlerCreateUser)

	mux.HandleFunc("GET /api/chirps", apiCfg.handlerGetAllChirps)
	mux.HandleFunc("GET /api/chirps/{chirpID}", apiCfg.handlerGetChirpByID)
	mux.HandleFunc("POST /api/chirps", apiCfg.handlerCreateChirp)

	mux.HandleFunc("GET /admin/metrics", apiCfg.handlerMetricsGet)
	mux.HandleFunc("POST /admin/reset", apiCfg.handlerResetGet)

	srv := &http.Server{
		Addr: ":" + port,
		Handler: mux,
	}
	
	log.Printf("Serving files from %s on port: %s\n", filepathRoot, port)
	log.Fatal(srv.ListenAndServe())
}







