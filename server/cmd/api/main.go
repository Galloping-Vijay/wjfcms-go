package main

import (
	"log"

	"wjfcm-go/internal/config"
	"wjfcm-go/internal/database"
	"wjfcm-go/internal/router"
)

func main() {
	cfg := config.Load()

	db, err := database.Open(cfg)
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}

	r := router.New(cfg, db)
	if err := r.Run(":" + cfg.App.Port); err != nil {
		log.Fatalf("run server: %v", err)
	}
}
