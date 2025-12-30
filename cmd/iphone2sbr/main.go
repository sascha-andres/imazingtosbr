package main

import (
	"log/slog"
	"os"
	"time"

	"github.com/sascha-andres/flag"

	"github.com/sascha-andres/imazingtosbr"
)

var (
	logLevel       int
	importFile     string
	collectionFile string
	tag            string
)

const (
	appPrefix = "IPHONE2SBR"
)

// initializeLogger initializes the logger
func initializeLogger(logLevel int) *slog.Logger {
	slogLevel, ok := map[int]slog.Level{
		0: slog.LevelWarn,
		1: slog.LevelInfo,
		2: slog.LevelDebug,
	}[logLevel]
	if !ok {
		slogLevel = slog.LevelDebug
	}
	handlerOpts := &slog.HandlerOptions{Level: slogLevel}
	logger := slog.New(slog.NewJSONHandler(os.Stderr, handlerOpts))
	slog.SetDefault(logger)
	return logger
}

// main is the entry point for the application
func main() {
	flag.IntVar(&logLevel, "log-level", 2, "Log level (0=warn, 1=info, 2=debug)")
	flag.StringVar(&importFile, "import-file", "testdata/Call History - 2025-12-07 07 00 00.csv", "Path to the file to import")
	flag.StringVar(&collectionFile, "collection-file", "testdata/calls.json", "Path to the collection file to append to")
	flag.StringVar(&tag, "tag", "", "Tag to apply to all imported calls")

	flag.SetEnvPrefix(appPrefix)
	flag.Parse()

	logger := initializeLogger(logLevel)

	start := time.Now()
	logger.Info("starting application")
	defer logger.Info("application stopped", "duration_ms", time.Since(start).Milliseconds())
	if err := run(logger, os.Args); err != nil {
		logger.Error("error running application", "err", err)
		os.Exit(1)
	}
}

// run runs the application
func run(logger *slog.Logger, _ []string) error {
	a, err := imazingtosbr.NewApplication(logger,
		imazingtosbr.WithCsvFile(importFile),
		imazingtosbr.WithCollectionFile(collectionFile),
		imazingtosbr.WithTag(tag))
	if err != nil {
		return err
	}
	sbrData, err := a.Convert()
	if err != nil {
		return err
	}
	if sbrData == nil {
		logger.Info("no data found")
		return nil
	}
	for _, call := range sbrData.GetCalls() {
		logger.Debug("call found", "call", call)
	}
	return a.Append(sbrData)
}
