package main

import (
	"flag"
	"log"

	"wjfcm-go/internal/applog"
	"wjfcm-go/internal/config"
	"wjfcm-go/internal/database"
	"wjfcm-go/internal/router"
)

func main() {
	envFile := flag.String("f", "", "env file path, default .env")
	flag.Parse()

	var cfg config.Config
	if *envFile != "" {
		cfg = config.Load(*envFile)
	} else {
		cfg = config.Load()
	}

	closeLog, err := applog.Configure(cfg)
	if err != nil {
		log.Fatalf("configure log: %v", err)
	}
	defer func() {
		if err := closeLog(); err != nil {
			log.Printf("close log: %v", err)
		}
	}()
	log.Printf("starting %s env=%s debug=%t log_channel=%s", cfg.App.Name, cfg.App.Env, cfg.App.Debug, cfg.Log.Channel)

	db, err := database.Open(cfg)
	if err != nil {
		log.Printf("connect database: %v", err)
		log.Printf("database is unavailable, only installer routes may work")
	}

	r := router.New(cfg, db)
	if err := r.Run(":" + cfg.App.Port); err != nil {
		log.Fatalf("run server: %v", err)
	}
}
