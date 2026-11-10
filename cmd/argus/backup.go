package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"

	"github.com/cragone/Super-Argus/internal/backup"
	"github.com/cragone/Super-Argus/internal/config"
	"github.com/cragone/Super-Argus/internal/db"
)

const usage = `usage: argus [command]

  (no command)       run the server
  backup <dir>       copy the data directory to <dir> while the server runs
                     (incremental: run it again to the same <dir> to update it)
  tui                process-compose dashboard tailing the argus/argus-agent
                     systemd logs side by side (needs process-compose on PATH;
                     read-only monitoring, not process control)
  restart <service>  ask the server to restart a systemd unit or pod via its
                     agent and wait for the result (admin; set ARGUS_EMAIL and
                     ARGUS_PASSWORD, and ARGUS_URL if not localhost:8080).
                     flags: -url, -target N (when there are several), -yes
`

// runCommand runs a subcommand and returns the process exit code.
func runCommand(name string, args []string, log *slog.Logger) int {
	switch name {
	case "backup":
		return runBackup(args, log)
	case "tui":
		return runTUI(args, log)
	case "restart":
		return runRestart(args, log)
	case "help", "-h", "--help":
		fmt.Print(usage)
		return 0
	}
	fmt.Fprintf(os.Stderr, "unknown command %q\n\n%s", name, usage)
	return 2
}

func runBackup(args []string, log *slog.Logger) int {
	if len(args) != 1 {
		fmt.Fprint(os.Stderr, usage)
		return 2
	}
	cfg, err := config.LoadTool()
	if err != nil {
		log.Error("config", "err", err)
		return 1
	}
	g, err := db.Open(cfg.DBDSN)
	if err != nil {
		log.Error("db open", "err", err)
		return 1
	}
	if sqlDB, err := g.DB(); err == nil {
		defer sqlDB.Close()
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	res, err := backup.Run(ctx, g, cfg.DataDir, args[0], cfg.ChunkStore == "local")
	if err != nil {
		log.Error("backup failed", "err", err)
		return 1
	}
	if res.ChunksNotCopiedReason != "" {
		log.Warn("backup complete, chunks not covered", "dest", args[0], "db_bytes", res.DBBytes,
			"wal_files", res.WALFiles, "reason", res.ChunksNotCopiedReason)
		return 0
	}
	log.Info("backup complete", "dest", args[0], "db_bytes", res.DBBytes, "wal_files", res.WALFiles,
		"chunks_copied", res.ChunksCopied, "chunks_unchanged", res.ChunksSkipped, "chunks_missing", res.MissingChunks)
	return 0
}
