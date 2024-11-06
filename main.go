package main

import (
	"embed"
	"io/fs"
	"log/slog"
	"os"

	_ "embed"

	"github.com/robherley/etherlighter/internal/config"
	"github.com/robherley/etherlighter/internal/device"
	"github.com/robherley/etherlighter/internal/server"
)

var (
	//go:embed web/*
	webFS embed.FS
)

func run() error {
	cfg, err := config.Load()
	if err != nil {
		cfg.Help(err)
	}

	log := slog.With("device", cfg.DeviceAddr, "listen", cfg.ListenAddr)

	log.Info("connecting")
	client, err := device.Connect(cfg)
	if err != nil {
		return err
	}
	defer client.Close()
	log.Info("connected!")

	var files fs.FS
	if cfg.DevMode {
		log.Warn("development mode enabled, web assets will be reloaded on every request")
		files = os.DirFS("web")
	} else {
		files, err = fs.Sub(webFS, "web")
		if err != nil {
			return err
		}
	}

	log.Info("starting server")
	server, err := server.New(cfg, files, client)
	if err != nil {
		return err
	}

	return server.ListenAndServe()
}

func main() {
	if err := run(); err != nil {
		slog.Error("failed", "error", err)
		os.Exit(1)
	}
}
