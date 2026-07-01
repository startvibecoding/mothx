package main

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/startvibecoding/vibecoding/internal/platform"
	"github.com/startvibecoding/vibecoding/internal/stats"
)

func newStatsCommand() *cobra.Command {
	flags := &statsFlags{}
	cmd := &cobra.Command{
		Use:   "stats",
		Short: "Start the stats dashboard web server",
		Long:  "Start a web server that displays usage statistics (tokens, requests, cost) with charts.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatsServer(flags)
		},
	}
	registerStatsFlags(cmd.Flags(), flags)
	return cmd
}

type statsFlags struct {
	addr   string
	dbPath string
}

func registerStatsFlags(fs *pflag.FlagSet, flags *statsFlags) {
	fs.StringVar(&flags.addr, "addr", "127.0.0.1:7878", "Listen address for the stats web server")
	fs.StringVar(&flags.dbPath, "db", "", "Path to sessions.db (default: <config-dir>/sessions/sessions.db)")
}

func runStatsServer(flags *statsFlags) error {
	dbPath := flags.dbPath
	if dbPath == "" {
		dbPath = filepath.Join(platform.SessionDir(), "sessions.db")
	}

	db, err := stats.Open(dbPath)
	if err != nil {
		return fmt.Errorf("open stats database: %w", err)
	}
	defer db.Close()

	server := stats.NewServer(db, flags.addr)
	return server.Start()
}
