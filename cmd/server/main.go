package main

import (
	"log"
	"os"

	"github.com/tarikogut/ai-callcenter-orchestrator/pkg/api"
	"github.com/tarikogut/ai-callcenter-orchestrator/pkg/db"
)

func main() {
	log.Println("Starting AI Call Center Orchestrator...")

	database, err := db.InitDB("")
	if err != nil {
		log.Fatalf("Fatal: Database initialization failed: %v", err)
	}

	app := api.SetupApp(database)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server listening on port %s", port)
	if err := app.Listen(":" + port); err != nil {
		log.Fatalf("Fatal: Server listen error: %v", err)
	}
}
