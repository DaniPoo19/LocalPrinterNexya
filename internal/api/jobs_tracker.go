package api

import (
	"fmt"
	"sync"
	"time"

	"local-printer-nexya/internal/utils"
)

type PrintJobRecord struct {
	ID          string    `json:"id"`
	Time        string    `json:"time"`
	OrderCode   string    `json:"order_code"`
	PrinterName string    `json:"printer_name"`
	Copies      int       `json:"copies"`
	Total       float64   `json:"total"`
	Success     bool      `json:"success"`
	Error       string    `json:"error,omitempty"`
	RawPayload  []byte    `json:"-"`
	CreatedAt   time.Time `json:"-"`
}

var (
	jobsMutex sync.RWMutex
	jobsList  []PrintJobRecord
	maxJobs   = 50
)

func AddJobRecord(record PrintJobRecord) {
	jobsMutex.Lock()
	defer jobsMutex.Unlock()

	if record.ID == "" {
		record.ID = fmt.Sprintf("job_%d", time.Now().UnixNano())
	}
	if record.Time == "" {
		record.Time = utils.FormatSpanishTimeOnly12h(utils.GetBogotaTime())
	}
	record.CreatedAt = time.Now()

	// Prepend to show most recent first
	jobsList = append([]PrintJobRecord{record}, jobsList...)
	if len(jobsList) > maxJobs {
		jobsList = jobsList[:maxJobs]
	}
}

func GetJobRecords() []PrintJobRecord {
	jobsMutex.RLock()
	defer jobsMutex.RUnlock()

	result := make([]PrintJobRecord, len(jobsList))
	copy(result, jobsList)
	return result
}

func GetJobRecordByID(id string) *PrintJobRecord {
	jobsMutex.RLock()
	defer jobsMutex.RUnlock()

	for _, j := range jobsList {
		if j.ID == id {
			copyJob := j
			return &copyJob
		}
	}
	return nil
}

func ClearJobRecords() {
	jobsMutex.Lock()
	defer jobsMutex.Unlock()
	jobsList = []PrintJobRecord{}
}
