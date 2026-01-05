package imazingtosbr

import (
	"encoding/csv"
	"errors"
	"log/slog"
	"os"
	"time"

	"github.com/sascha-andres/reuse"
	"github.com/sascha-andres/sbrdata/v2"
)

var (
	// ErrImportFileDoesNotExist is returned when the import file does not exist
	ErrImportFileDoesNotExist = errors.New("import file does not exist")

	// headerIndexMapCall maps the header names to their index in the CSV file for calls
	headerIndexMapCall = map[string]int{
		"Call Type": 0,
		"Date":      1,
		"Duration":  2,
		"Number":    3,
		"Contact":   4,
		"Location":  5,
		"Service":   6,
	}

	// headerIndexMapMessages maps the header names to their index in the CSV file for messages
	headerIndexMapMessages = map[string]int{
		"Chat Session":    0,
		"Message Date":    1,
		"Delivered Date":  2,
		"Read Date":       3,
		"Edited Date":     4,
		"Service":         5,
		"Type":            6,
		"Sender ID":       7,
		"Sender Name":     8,
		"Status":          9,
		"Replying to":     10,
		"Subject":         11,
		"Text":            12,
		"Attachment":      13,
		"Attachment type": 14,
	}
)

const (
	phonePrefix      = "Phone: "
	callTypeOutgoing = "Outgoing"
	callTypeIncoming = "Incoming"
)

// Application represents the main application component
type Application struct {
	// Logger instance for logging application events
	l *slog.Logger
	// File to import
	fileToImport string
	// Collection file to append to
	collectionFile string
	// Tag to apply to all imported calls
	tag string
}

// AppendCalls adds the calls to the collection file
func (a *Application) AppendCalls(calls *sbrdata.Calls) error {
	var err error
	collection := &sbrdata.Collection{
		Key:   "",
		Calls: make([]sbrdata.Call, 0),
		Sms:   make([]sbrdata.SMS, 0),
		Mms:   make([]sbrdata.MMS, 0),
	}
	if reuse.FileExists(a.collectionFile) {
		collection, err = sbrdata.LoadCollection(a.collectionFile)
		if err != nil {
			return err
		}
	}
	err = collection.AddCalls(calls)
	if err != nil {
		return err
	}
	return collection.Save(a.collectionFile)
}

// AppendMessages adds the messages to the collection file
func (a *Application) AppendMessages(messages *sbrdata.Messages) error {
	var err error
	collection := &sbrdata.Collection{
		Key:   "",
		Calls: make([]sbrdata.Call, 0),
		Sms:   make([]sbrdata.SMS, 0),
		Mms:   make([]sbrdata.MMS, 0),
	}
	if reuse.FileExists(a.collectionFile) {
		collection, err = sbrdata.LoadCollection(a.collectionFile)
		if err != nil {
			return err
		}
	}
	err = collection.AddMessages(messages)
	if err != nil {
		return err
	}
	return collection.Save(a.collectionFile)
}

type FileType uint

const (
	UnknownFile FileType = iota
	CallHistoryFile
	MessageHistoryFile
)

// Convert converts the CSV file to SBR data
func (a *Application) Convert() (any, FileType, error) {
	start := time.Now()
	a.l.Debug("converting file", "file", a.fileToImport)
	defer a.l.Debug("conversion finished", "duration_ms", time.Since(start).Milliseconds())

	file, err := os.Open(a.fileToImport)
	if err != nil {
		return nil, UnknownFile, err
	}
	defer func() {
		err := file.Close()
		if err != nil {
			a.l.Error("error closing file", "err", err)
		}
	}()

	csvIn := csv.NewReader(file)

	// print header in debug mode in case anything changes
	header, err := csvIn.Read()
	if err != nil {
		return nil, UnknownFile, err
	}
	for _, h := range header {
		a.l.Debug("header", "header", h, "map", headerIndexMapCall[h])
	}

	csvIn.ReuseRecord = true

	if header[0] == "Call type" {
		// it is a call history file
		return a.transformCallData(csvIn)
	}
	if header[0] == "Chat Session" {
		return a.transformMessageData(csvIn)
	}

	return nil, UnknownFile, errors.New("unsupported file format")
}

// str2Ptr converts a string to a pointer
func str2Ptr(s string) *string { return &s }

// ApplicationOption represents an option for the Application
type ApplicationOption func(*Application) error

// WithCollectionFile sets the collection file to append to
func WithCollectionFile(file string) ApplicationOption {
	return func(app *Application) error {
		app.collectionFile = file
		return nil
	}
}

// WithCsvFile sets the file to import
func WithCsvFile(fileToImport string) ApplicationOption {
	return func(app *Application) error {
		if !reuse.FileExists(fileToImport) {
			return ErrImportFileDoesNotExist
		}
		app.fileToImport = fileToImport
		return nil
	}
}

// WithTag sets the tag to apply to all imported calls
func WithTag(tag string) ApplicationOption {
	return func(app *Application) error {
		app.tag = tag
		return nil
	}
}

// NewApplication creates a new Application
func NewApplication(l *slog.Logger, opts ...ApplicationOption) (*Application, error) {
	app := &Application{l: l}
	for _, opt := range opts {
		if err := opt(app); err != nil {
			return nil, err
		}
	}
	return app, nil
}
