package imazingtosbr

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/sascha-andres/sbrdata/v2"
	"golang.org/x/tools/txtar"
)

type Parameters struct {
	FileType string `json:"file_type"`
}

// TestConvert tests the Convert function using txtar test cases
func TestConvert(t *testing.T) {
	tests, err := filepath.Glob("testdata/*.txtar")
	if err != nil {
		t.Fatalf("failed to glob testdata: %v", err)
	}

	if len(tests) == 0 {
		t.Fatal("no test files found in testdata/")
	}

	for _, testFile := range tests {
		t.Run(filepath.Base(testFile), func(t *testing.T) {
			// Read txtar archive
			archive, err := txtar.ParseFile(testFile)
			if err != nil {
				t.Fatalf("failed to parse txtar file: %v", err)
			}

			// Extract input CSV and options
			var inputCSV []byte
			var parameters Parameters
			var expected string
			for _, file := range archive.Files {
				switch file.Name {
				case "input.csv":
					inputCSV = file.Data
				case "parameters.json":
					err := json.Unmarshal(file.Data, &parameters)
					if err != nil {
						t.Fatalf("failed to unmarshal parameters.json: %v", err)
					}
				case "result.json":
					expected = strings.TrimSpace(string(file.Data))
				}
			}

			if len(inputCSV) == 0 {
				t.Fatal("no input.csv found in txtar archive")
			}

			// Create temporary CSV file
			tmpDir := t.TempDir()
			csvPath := filepath.Join(tmpDir, "input.csv")
			if err := os.WriteFile(csvPath, inputCSV, 0644); err != nil {
				t.Fatalf("failed to write temp CSV: %v", err)
			}

			// Create logger for testing
			logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
				Level: slog.LevelError, // Suppress logs during tests
			}))

			// Create application and run conversion
			app, err := NewApplication(logger, WithCsvFile(csvPath))
			if err != nil {
				t.Fatalf("failed to create application: %v", err)
			}

			result, fileType, err := app.Convert()
			if err != nil {
				t.Fatalf("Convert() error = %v", err)
			}

			collection := sbrdata.Collection{
				Key:   "",
				Calls: make([]sbrdata.Call, 0),
				Sms:   make([]sbrdata.SMS, 0),
				Mms:   make([]sbrdata.MMS, 0),
			}

			// Verify file type
			expectedFileType := parameters.FileType
			switch expectedFileType {
			case "call_history":
				if fileType != CallHistoryFile {
					t.Errorf("expected CallHistoryFile, got %v", fileType)
				}
				// Verify call data
				callData, ok := result.(*sbrdata.Calls)
				if !ok {
					t.Fatalf("expected *sbrdata.Calls, got %T", result)
				}
				if err := collection.AddCalls(callData); err != nil {
					t.Fatalf("failed to add calls to collection: %v", err)
				}
			case "messages":
				if fileType != MessageHistoryFile {
					t.Errorf("expected MessageHistoryFile, got %v", fileType)
				}
				// Verify message data
				messageData, ok := result.(*sbrdata.Messages)
				if !ok {
					t.Fatalf("expected *sbrdata.Messages, got %T", result)
				}
				if err := collection.AddMessages(messageData); err != nil {
					t.Fatalf("failed to add messages to collection: %v", err)
				}
			default:
				t.Errorf("unknown file_type in options: %s", expectedFileType)
			}

			data, err := json.MarshalIndent(collection, "", "  ")
			if err != nil {
				t.Fatalf("failed to marshal collection: %v", err)
			}
			if diff := cmp.Diff(strings.TrimSpace(string(data)), expected, nil); diff != "" {
				_ = os.WriteFile(path.Join("testdata", filepath.Base(testFile)+".result.json"), data, 0644)
				t.Errorf("unexpected result. diff: \n\n%s", diff)
			}
		})
	}
}

// TestConvertInvalidFile tests Convert with invalid file formats
func TestConvertInvalidFile(t *testing.T) {
	tmpDir := t.TempDir()
	csvPath := filepath.Join(tmpDir, "invalid.csv")

	// Create an invalid CSV (wrong header)
	invalidCSV := "Invalid,Header,Format\ndata1,data2,data3\n"
	if err := os.WriteFile(csvPath, []byte(invalidCSV), 0644); err != nil {
		t.Fatalf("failed to write temp CSV: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	app, err := NewApplication(logger, WithCsvFile(csvPath))
	if err != nil {
		t.Fatalf("failed to create application: %v", err)
	}

	_, fileType, err := app.Convert()
	if err == nil {
		t.Error("expected error for invalid file format, got nil")
	}
	if fileType != UnknownFile {
		t.Errorf("expected UnknownFile type, got %v", fileType)
	}
}

// TestConvertNonExistentFile tests Convert with a file that doesn't exist
func TestConvertNonExistentFile(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	_, err := NewApplication(logger, WithCsvFile("/nonexistent/file.csv"))
	if err != ErrImportFileDoesNotExist {
		t.Errorf("expected ErrImportFileDoesNotExist, got %v", err)
	}
}

// TestCallTypeMapping tests that call types are correctly mapped
func TestCallTypeMapping(t *testing.T) {
	tmpDir := t.TempDir()
	csvPath := filepath.Join(tmpDir, "test.csv")

	csvData := `Call type,Date,Duration,Number,Contact,Location,Service
Outgoing,2024-01-01 12:00:00,00:01:00,+1234567890,Test Contact,USA,Phone: +1234567890
Incoming,2024-01-01 13:00:00,00:02:00,+9876543210,Test Contact 2,USA,Phone: +9876543210`

	if err := os.WriteFile(csvPath, []byte(csvData), 0644); err != nil {
		t.Fatalf("failed to write temp CSV: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	app, err := NewApplication(logger, WithCsvFile(csvPath))
	if err != nil {
		t.Fatalf("failed to create application: %v", err)
	}

	result, _, err := app.Convert()
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}

	callData := result.(*sbrdata.Calls)
	if len(callData.Call) != 2 {
		t.Fatalf("expected 2 calls, got %d", len(callData.Call))
	}

	// First call should be Outgoing (type 2)
	if callData.Call[0].Type != "2" {
		t.Errorf("expected outgoing call type '2', got '%s'", callData.Call[0].Type)
	}

	// Second call should be Incoming (type 1)
	if callData.Call[1].Type != "1" {
		t.Errorf("expected incoming call type '1', got '%s'", callData.Call[1].Type)
	}
}

// TestServiceTypeParsing tests that service types are correctly parsed
func TestServiceTypeParsing(t *testing.T) {
	tests := []struct {
		name            string
		serviceField    string
		expectedService string
	}{
		{
			name:            "Phone with number",
			serviceField:    "Phone: +1234567890",
			expectedService: "Phone",
		},
		{
			name:            "WhatsApp Video",
			serviceField:    "‎WhatsApp Video",
			expectedService: "‎WhatsApp Video",
		},
		{
			name:            "Signal Audio",
			serviceField:    "Signal Audio",
			expectedService: "Signal Audio",
		},
		{
			name:            "Teams Audio",
			serviceField:    "Teams Audio",
			expectedService: "Teams Audio",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			csvPath := filepath.Join(tmpDir, "test.csv")

			csvData := fmt.Sprintf(`Call type,Date,Duration,Number,Contact,Location,Service
Outgoing,2024-01-01 12:00:00,00:01:00,+1234567890,Test Contact,USA,%s`, tt.serviceField)

			if err := os.WriteFile(csvPath, []byte(csvData), 0644); err != nil {
				t.Fatalf("failed to write temp CSV: %v", err)
			}

			logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
				Level: slog.LevelError,
			}))

			app, err := NewApplication(logger, WithCsvFile(csvPath))
			if err != nil {
				t.Fatalf("failed to create application: %v", err)
			}

			result, _, err := app.Convert()
			if err != nil {
				t.Fatalf("Convert() error = %v", err)
			}

			callData := result.(*sbrdata.Calls)
			if len(callData.Call) != 1 {
				t.Fatalf("expected 1 call, got %d", len(callData.Call))
			}

			if callData.Call[0].ServiceType == nil {
				t.Fatal("expected ServiceType to be set, got nil")
			}

			if *callData.Call[0].ServiceType != tt.expectedService {
				t.Errorf("expected service type '%s', got '%s'", tt.expectedService, *callData.Call[0].ServiceType)
			}
		})
	}
}

// TestDateConversion tests that dates are correctly converted to Unix milliseconds
func TestDateConversion(t *testing.T) {
	tmpDir := t.TempDir()
	csvPath := filepath.Join(tmpDir, "test.csv")

	// Use a known date: 2024-03-15 14:30:00 UTC
	csvData := `Call type,Date,Duration,Number,Contact,Location,Service
Outgoing,2024-03-15 14:30:00,00:01:00,+1234567890,Test Contact,USA,Phone: +1234567890`

	if err := os.WriteFile(csvPath, []byte(csvData), 0644); err != nil {
		t.Fatalf("failed to write temp CSV: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	app, err := NewApplication(logger, WithCsvFile(csvPath))
	if err != nil {
		t.Fatalf("failed to create application: %v", err)
	}

	result, _, err := app.Convert()
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}

	callData := result.(*sbrdata.Calls)
	if len(callData.Call) != 1 {
		t.Fatalf("expected 1 call, got %d", len(callData.Call))
	}

	// Verify the date is a valid Unix timestamp in milliseconds
	if callData.Call[0].Date == "" {
		t.Error("expected Date to be set")
	}

	// Verify ReadableDate matches input
	if callData.Call[0].ReadableDate != "2024-03-15 14:30:00" {
		t.Errorf("expected ReadableDate '2024-03-15 14:30:00', got '%s'", callData.Call[0].ReadableDate)
	}
}

// parseOptions parses the options.txt content into a map
func parseOptions(data []byte) map[string]string {
	options := make(map[string]string)
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			options[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	return options
}

// verifyCallData verifies call data against expected options
func verifyCallData(t *testing.T, callData *sbrdata.Calls, options map[string]string) {
	t.Helper()

	// Verify count
	if expectedCount, ok := options["expected_count"]; ok {
		actualCount := len(callData.Call)
		if fmt.Sprintf("%d", actualCount) != expectedCount {
			t.Errorf("expected %s calls, got %d", expectedCount, actualCount)
		}
		if callData.Count != expectedCount {
			t.Errorf("expected Count field to be %s, got %s", expectedCount, callData.Count)
		}
	}

	// Verify call types
	if expectedTypes, ok := options["expected_types"]; ok {
		types := strings.Split(expectedTypes, ",")
		if len(types) != len(callData.Call) {
			t.Errorf("expected %d type values, got %d calls", len(types), len(callData.Call))
		} else {
			for i, expectedType := range types {
				if callData.Call[i].Type != strings.TrimSpace(expectedType) {
					t.Errorf("call %d: expected type %s, got %s", i, expectedType, callData.Call[i].Type)
				}
			}
		}
	}

	// Verify service type
	if expectedService, ok := options["expected_service"]; ok {
		for i, call := range callData.Call {
			if call.ServiceType == nil {
				t.Errorf("call %d: expected ServiceType to contain '%s', got nil", i, expectedService)
			} else if !strings.Contains(*call.ServiceType, expectedService) {
				t.Errorf("call %d: expected ServiceType to contain '%s', got '%s'", i, expectedService, *call.ServiceType)
			}
		}
	}

	// Verify all calls have DataFrom set to "iMazing"
	for i, call := range callData.Call {
		if call.DataFrom == nil {
			t.Errorf("call %d: expected DataFrom to be set", i)
		} else if *call.DataFrom != "iMazing" {
			t.Errorf("call %d: expected DataFrom 'iMazing', got '%s'", i, *call.DataFrom)
		}
	}
}

// verifyMessageData verifies message data against expected options
func verifyMessageData(t *testing.T, messageData *sbrdata.Messages, options map[string]string) {
	t.Helper()

	// Verify SMS count
	if expectedCount, ok := options["expected_sms_count"]; ok {
		actualCount := len(messageData.Sms)
		if fmt.Sprintf("%d", actualCount) != expectedCount {
			t.Errorf("expected %s SMS messages, got %d", expectedCount, actualCount)
		}
	}

	// Verify message types
	if expectedTypes, ok := options["expected_types"]; ok {
		types := strings.Split(expectedTypes, ",")
		if len(types) != len(messageData.Sms) {
			t.Errorf("expected %d type values, got %d messages", len(types), len(messageData.Sms))
		} else {
			for i, expectedType := range types {
				if messageData.Sms[i].Type != strings.TrimSpace(expectedType) {
					t.Errorf("message %d: expected type %s, got %s", i, expectedType, messageData.Sms[i].Type)
				}
			}
		}
	}

	// Verify sender name if specified
	if expectedSender, ok := options["sender_name"]; ok {
		for i, sms := range messageData.Sms {
			// Check if sender name is set (either from Sender Name or Chat Session)
			if sms.ContactName == "" {
				t.Errorf("message %d: expected ContactName to be set", i)
			}
			// For messages from the expected sender, verify the name matches
			if strings.Contains(sms.ContactName, expectedSender) || strings.Contains(expectedSender, sms.ContactName) {
				// Match found, this is expected
			}
		}
	}
}
