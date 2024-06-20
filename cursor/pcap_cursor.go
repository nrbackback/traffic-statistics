package cursor

import (
	"traffic-statistics/pkg/log"
)

type PcapCursor interface {
	AddOneUploadedRecord(filename string) error
	MarkFileAsUploaded(filename string) error
	IsFileUploaded(filename string) (bool, error)
	UploadedRecordCount(filename string) (int64, error)
	Stop()
}

func BuildPcapCursor(cursorType string, config map[string]interface{}) PcapCursor {
	if cursorType == sqliteType {
		return newSqliteHandler(config)
	}
	if cursorType == FileType {
		return newFileHandler(config)
	}
	log.Fatalf("invalid cursor type", "type", cursorType)
	return nil
}
