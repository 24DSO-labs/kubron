// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package audit

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

// Форматы выгрузки.
const (
	FormatCSV  = "csv"
	FormatJSON = "json"
)

var errUnknownFormat = errors.New("unknown export format: use csv or json")

// Export выгружает записи, подходящие под фильтр (до ExportLimit штук, от новых к старым), в CSV или JSON.
// truncated — подходящих записей больше, чем выгружено.
func (l *Logger) Export(f Filter, format string) (data []byte, count int, truncated bool, err error) {
	if format != FormatCSV && format != FormatJSON {
		return nil, 0, false, errUnknownFormat
	}
	page, err := l.query(f, ExportLimit)
	if err != nil {
		return nil, 0, false, err
	}
	if format == FormatJSON {
		data, err = json.MarshalIndent(page.Events, "", "  ")
		return data, len(page.Events), page.Truncated, err
	}
	data, err = eventsCSV(page.Events)
	return data, len(page.Events), page.Truncated, err
}

var csvHeader = []string{"time", "user", "role", "action", "result", "detail", "source", "team_id", "cluster_id"}

func eventsCSV(events []Event) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteString("\xEF\xBB\xBF") // BOM: Excel открывает UTF-8 без «кракозябр»
	w := csv.NewWriter(&buf)
	if err := w.Write(csvHeader); err != nil {
		return nil, err
	}
	for _, e := range events {
		row := []string{
			e.Time.UTC().Format(time.RFC3339), e.User, e.Role, e.Action, e.Result, e.Detail, e.Source, e.TeamID, e.ClusterID,
		}
		for i := range row {
			row[i] = csvSafe(row[i])
		}
		if err := w.Write(row); err != nil {
			return nil, err
		}
	}
	w.Flush()
	return buf.Bytes(), w.Error()
}

// csvSafe защищает от инъекции формул: ячейка, начинающаяся с = + - @ или управляющего символа, в Excel и
// LibreOffice исполняется как формула. Значения в журнале (логин, детали) задают пользователи, поэтому
// такие ячейки получают ведущий апостроф.
func csvSafe(s string) string {
	if s == "" {
		return s
	}
	if strings.ContainsRune("=+-@\t\r", rune(s[0])) {
		return "'" + s
	}
	return s
}
