package store

import (
	"database/sql"
	"fmt"
)

type Store struct{ DB *sql.DB }

func New(db *sql.DB) *Store { return &Store{DB: db} }

func (s *Store) Rows(query string, args ...any) ([]map[string]any, error) {
	rows, err := s.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	result := make([]map[string]any, 0)
	for rows.Next() {
		values := make([]any, len(columns))
		pointers := make([]any, len(columns))
		for i := range values {
			pointers[i] = &values[i]
		}
		if err := rows.Scan(pointers...); err != nil {
			return nil, err
		}
		item := make(map[string]any, len(columns))
		for i, value := range values {
			if bytes, ok := value.([]byte); ok {
				item[columns[i]] = string(bytes)
			} else {
				item[columns[i]] = value
			}
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read rows: %w", err)
	}
	return result, nil
}
