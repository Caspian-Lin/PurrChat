package main

import (
	"context"
	"fmt"
	"log"
	"os"

	migrate "github.com/Caspian-Lin/PurrChat/packages/db-migrate"
	"purr-chat-server/pkg/config"
	"purr-chat-server/pkg/database"
)

var legacyVersions = map[string]string{
	"006_fix_conversation_message_function_permissions.sql": "006a",
	"006_message_created_at_ordering.sql":                   "006b",
}

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
		Service:        "backend",
		Dir:            "migrations",
		BaselineTable:  "users",
		LegacyVersions: legacyVersions,
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
	fmt.Printf("backend migrations %s completed\n", command)
}
