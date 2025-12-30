package imazingtosbr

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/sascha-andres/reuse"
	"github.com/sascha-andres/sbrdata/v2"
)

var (
	// ErrImportFileDoesNotExist is returned when the import file does not exist
	ErrImportFileDoesNotExist = errors.New("import file does not exist")

	// headerIndexMap maps the header names to their index in the CSV file
	headerIndexMap = map[string]int{
		"Call Type": 0,
		"Date":      1,
		"Duration":  2,
		"Number":    3,
		"Contact":   4,
		"Location":  5,
		"Service":   6,
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

// Append adds the calls to the collection file
func (a *Application) Append(calls *sbrdata.Calls) error {
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
	collection.SetVerbose()
	return collection.Save(a.collectionFile)
}

// Convert converts the CSV file to SBR data
func (a Application) Convert() (*sbrdata.Calls, error) {
	start := time.Now()
	a.l.Debug("converting file", "file", a.fileToImport)
	defer a.l.Debug("conversion finished", "duration_ms", time.Since(start).Milliseconds())

	file, err := os.Open(a.fileToImport)
	if err != nil {
		return nil, err
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
		return nil, err
	}
	for _, h := range header {
		a.l.Debug("header", "header", h, "map", headerIndexMap[h])
	}
	csvIn.ReuseRecord = true

	callData := sbrdata.Calls{
		Call:  make([]sbrdata.Call, 0),
		Count: "0",
	}

	for {
		record, err := csvIn.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		svc := ""
		if strings.Contains(record[headerIndexMap["Service"]], ":") {
			serviceData := strings.SplitN(record[headerIndexMap["Service"]], ":", 2)
			svc = serviceData[0]
		} else {
			svc = record[headerIndexMap["Service"]]
		}
		date := ""
		dt, err := time.Parse("2006-01-02 15:04:05", record[headerIndexMap["Date"]])
		if err != nil {
			return nil, err
		}
		date = fmt.Sprintf("%d", dt.UnixMilli())
		call := sbrdata.Call{
			ContactName:  record[headerIndexMap["Contact"]],
			Date:         date,
			ReadableDate: record[headerIndexMap["Date"]],
			Presentation: record[headerIndexMap["Contact"]],
			Duration:     record[headerIndexMap["Duration"]],
			DataFrom:     str2Ptr("iMazing"),
			ServiceType:  str2Ptr(svc),
			Number:       record[headerIndexMap["Number"]],
		}
		if record[headerIndexMap["Call Type"]] == callTypeOutgoing {
			call.Type = "2"
		} else {
			call.Type = "1"
		}
		callData.Call = append(callData.Call, call)
	}

	callData.Count = fmt.Sprintf("%d", len(callData.Call))
	return &callData, nil
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
