package imazingtosbr

import (
	"encoding/csv"
	"fmt"
	"io"
	"time"

	"github.com/sascha-andres/sbrdata/v2"
)

func (a *Application) transformMessageData(csvIn *csv.Reader) (any, FileType, error) {
	messageData := &sbrdata.Messages{
		Sms: make([]sbrdata.SMS, 0),
		Mms: make([]sbrdata.MMS, 0),
	}

	for {
		record, err := csvIn.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, CallHistoryFile, err
		}
		sms := sbrdata.SMS{}
		if record[headerIndexMapMessages["Type"]] == callTypeOutgoing {
			sms.Type = "2"
		} else {
			sms.Type = "1"
		}
		sms.ContactName = record[headerIndexMapMessages["Sender Name"]]
		if sms.ContactName == "" {
			sms.ContactName = record[headerIndexMapMessages["Chat Session"]]
		}
		date := ""
		dt, err := time.Parse("2006-01-02 15:04:05", record[headerIndexMapMessages["Message Date"]])
		if err != nil {
			return nil, CallHistoryFile, err
		}
		date = fmt.Sprintf("%d", dt.UnixMilli())
		sms.Subject = record[headerIndexMapMessages["Subject"]]
		sms.Body = record[headerIndexMapMessages["Text"]]
		sms.ReadableDate = record[headerIndexMapMessages["Message Date"]]
		sms.Date = date
		sms.Address = record[headerIndexMapMessages["Sender ID"]]
		sms.Status = record[headerIndexMapMessages["Status"]]
		messageData.Sms = append(messageData.Sms, sms)
	}

	return messageData, MessageHistoryFile, nil
}
