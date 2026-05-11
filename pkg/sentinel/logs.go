package sentinel

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"reflect"
)

const (
	logsPerRequest = 10
	tableName      = "WorkspacesOneLogs_CL"
)

func chunkLogs(logs interface{}, chunkSize int) ([]interface{}, error) {
	logsValue := reflect.ValueOf(logs)
	if logsValue.Kind() != reflect.Slice {
		return nil, fmt.Errorf("logs must be a slice")
	}

	chunks := make([]interface{}, 0)
	for i := 0; i < logsValue.Len(); i += chunkSize {
		end := i + chunkSize
		if end > logsValue.Len() {
			end = logsValue.Len()
		}

		chunks = append(chunks, logsValue.Slice(i, end).Interface())
	}

	return chunks, nil
}

func (s *Sentinel) SendLogs(ctx context.Context, l *logrus.Logger, endpoint, ruleID, streamName string, logs interface{}) error {
	logger := l.WithField("module", "sentinel_logs")
	logsValue := reflect.ValueOf(logs)
	if logsValue.Kind() != reflect.Slice {
		return fmt.Errorf("logs must be a slice")
	}

	logger.WithField("table_name", tableName).WithField("total", logsValue.Len()).Info("shipping logs")

	chunkedLogs, err := chunkLogs(logs, logsPerRequest)
	if err != nil {
		return err
	}
	for i, logsChunk := range chunkedLogs {
		l.WithField("progress", fmt.Sprintf("%d/%d", i+1, len(chunkedLogs))).Debug("ingesting log chunks")

		if reflect.ValueOf(logsChunk).Len() == 0 {
			l.Warn("processing empty chunk")
			continue
		}

		if err := s.IngestLog(ctx, endpoint, ruleID, streamName, logsChunk); err != nil {
			return fmt.Errorf("could not ingest log: %v", err)
		}
	}

	//

	logger.WithField("table_name", tableName).Info("shipped logs")

	return nil
}
