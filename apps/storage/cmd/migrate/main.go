package main

import (
	"context"
	"fmt"
	"log"
	"os"

	migrate "github.com/Caspian-Lin/PurrChat/packages/db-migrate"
	"purr-chat-storage/pkg/config"
	"purr-chat-storage/pkg/database"
)

func main() {
	command := "up"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}

	cfg := config.Load()
	if err := database.Init(config.GetDSN(&cfg.DB)); err != nil {
		log.Fatalf("connect to database: %v", err)
	}
	defer database.Close()

	runner := migrate.NewRunner(migrate.Config{
		Service:       "storage",
		Dir:           "migrations",
		BaselineTable: "file_metadata",
	})

	var err error
	switch command {
	case "up":
		err = runner.Run(context.Background(), database.GetPool())
	case "baseline":
		err = runner.Baseline(context.Background(), database.GetPool())
	default:
		log.Fatal("usage: migrate [up|baseline]")
	}
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("storage migrations %s completed\n", command)
}
