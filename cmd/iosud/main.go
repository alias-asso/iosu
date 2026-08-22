// Command iosud serves the contest website.
package main

import (
	"context"
	"flag"
	"log"

	"github.com/alias-asso/iosu/internal/app"
	"github.com/alias-asso/iosu/internal/config"
	"github.com/alias-asso/iosu/internal/store"
	"github.com/alias-asso/iosu/internal/web"
)

func main() {
	configPath := flag.String("c", config.DefaultPath, "config file path")
	flag.Parse()

	cfg, err := config.Parse(*configPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	db, err := store.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer db.Close()

	a := app.New(db, cfg.DataDir)
	ctx := context.Background()

	if created, err := a.EnsureAdmin(ctx, cfg.DefaultAdminPassword); err != nil {
		log.Fatalf("creating the admin account: %v", err)
	} else if created {
		log.Println(`created the "admin" account with the password from the config file; change it with: iosu user passwd -username admin`)
	}
	if created, err := a.EnsureSiteConfig(ctx); err != nil {
		log.Fatalf("creating the site config: %v", err)
	} else if created {
		log.Println("wrote placeholder site content; edit it with: iosu config update")
	}

	srv, err := web.New(a, cfg)
	if err != nil {
		log.Fatalf("server: %v", err)
	}
	log.Fatal(srv.Start(cfg.ServerPort))
}
