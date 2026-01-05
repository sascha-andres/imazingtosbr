package imazingtosbr

import (
	"encoding/csv"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/sascha-andres/sbrdata/v2"
)

// transformCallData reads call data from a CSV reader and transforms it into an sbrdata.Calls structure.
func (a *Application) transformCallData(csvIn *csv.Reader) (*sbrdata.Calls, FileType, error) {
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
			return nil, CallHistoryFile, err
		}
		svc := ""
		if strings.Contains(record[headerIndexMapCall["Service"]], ":") {
			serviceData := strings.SplitN(record[headerIndexMapCall["Service"]], ":", 2)
			svc = serviceData[0]
		} else {
			svc = record[headerIndexMapCall["Service"]]
		}
		date := ""
		dt, err := time.Parse("2006-01-02 15:04:05", record[headerIndexMapCall["Date"]])
		if err != nil {
			return nil, CallHistoryFile, err
		}
		date = fmt.Sprintf("%d", dt.UnixMilli())
		call := sbrdata.Call{
			ContactName:  record[headerIndexMapCall["Contact"]],
			Date:         date,
			ReadableDate: record[headerIndexMapCall["Date"]],
			Presentation: record[headerIndexMapCall["Contact"]],
			Duration:     record[headerIndexMapCall["Duration"]],
			DataFrom:     str2Ptr("iMazing"),
			ServiceType:  str2Ptr(svc),
			Number:       record[headerIndexMapCall["Number"]],
		}
		if record[headerIndexMapCall["Call Type"]] == callTypeOutgoing {
			call.Type = "2"
		} else {
			call.Type = "1"
		}
		callData.Call = append(callData.Call, call)
	}

	callData.Count = fmt.Sprintf("%d", len(callData.Call))
	return &callData, CallHistoryFile, nil
}
