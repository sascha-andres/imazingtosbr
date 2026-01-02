package main

import (
	"errors"
	"log/slog"
	"os"
	"time"

	"github.com/sascha-andres/reuse/flag"
	"github.com/sascha-andres/sbrdata/v2"

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
	start := time.Now()

	flag.SetEnvPrefix(appPrefix)
	flag.IntVar(&logLevel, "log-level", 2, "Log level (0=warn, 1=info, 2=debug)")
	flag.StringVar(&importFile, "import-file", "", "Path to the file to import")
	flag.StringVar(&collectionFile, "collection-file", "", "Path to the collection file to append to")
	flag.StringVar(&tag, "tag", "", "Tag to apply to all imported calls")
	flag.Parse()

	logger := initializeLogger(logLevel)
	logger.Info("starting application")
	defer logger.Info("application stopped", "duration_ms", time.Since(start).Milliseconds())

	if logLevel == 2 {
		logger.Info("input", "import-file", importFile, "collection-file", collectionFile, "tag", tag, "args", os.Args[1:])
	}

	if err := run(logger, os.Args); err != nil {
		logger.Error("error running application", "err", err, "duration_ms", time.Since(start).Milliseconds())
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
	sbrData, fileType, err := a.Convert()
	if err != nil {
		return err
	}
	logger.Info("converted file", "file_type", fileType)
	if sbrData == nil {
		logger.Info("no data found")
		return nil
	}
	if fileType == imazingtosbr.CallHistoryFile {
		callData := sbrData.(*sbrdata.Calls)
		for _, call := range callData.GetCalls() {
			logger.Debug("call found", "call", call)
		}
		return a.AppendCalls(callData)
	}
	// TODO handle SMS/MMS
	return errors.New("unsupported file type")
}
