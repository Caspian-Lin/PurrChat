package main

import (
	"context"
	"flag"
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
	flags := flag.NewFlagSet("migrate "+command, flag.ExitOnError)
	adminPassword := flags.String("admin-password", "", "database administrator password (uses DB_ADMIN_USER and SET ROLE DB_USER)")
	_ = flags.Parse(os.Args[2:])

	cfg := config.Load()
	dsn := config.GetDSN(&cfg.DB)
	if *adminPassword != "" {
		var err error
		dsn, err = migrate.AdminRoleDSN(migrate.AdminRoleConfig{
			Host:          envOr("DB_ADMIN_HOST", cfg.DB.Host),
			Port:          envOr("DB_ADMIN_PORT", cfg.DB.Port),
			Database:      cfg.DB.Name,
			AdminUser:     envOr("DB_ADMIN_USER", "postgres"),
			AdminPassword: *adminPassword,
			Role:          cfg.DB.User,
		})
		if err != nil {
			log.Fatal(err)
		}
	}
	if err := database.Init(dsn); err != nil {
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
		log.Fatal("usage: migrate [up|baseline] [--admin-password PASSWORD]")
	}
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("backend migrations %s completed\n", command)
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
