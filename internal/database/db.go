package database

import (
    "context"
    "database/sql"
    "fmt"
    "strings"
)

type DB struct {
    *sql.DB
}

func New(db *sql.DB) *DB {
    return &DB{db}
}

func (db *DB) ExecSafe(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
    stmt, err := db.PrepareContext(ctx, query)
    if err != nil {
        return nil, fmt.Errorf("prepare statement: %w", err)
    }
    defer stmt.Close()

    result, err := stmt.ExecContext(ctx, args...)
    if err != nil {
        return nil, fmt.Errorf("execute statement: %w", err)
    }

    return result, nil
}

func (db *DB) QuerySafe(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
    stmt, err := db.PrepareContext(ctx, query)
    if err != nil {
        return nil, fmt.Errorf("prepare statement: %w", err)
    }
    defer stmt.Close()

    rows, err := stmt.QueryContext(ctx, args...)
    if err != nil {
        return nil, fmt.Errorf("execute query: %w", err)
    }

    return rows, nil
}

func (db *DB) QueryRowSafe(ctx context.Context, query string, args ...interface{}) *sql.Row {
    stmt, err := db.PrepareContext(ctx, query)
    if err != nil {
        return db.QueryRowContext(ctx, "SELECT 1 WHERE false") // Return empty row on error
    }
    defer stmt.Close()

    return stmt.QueryRowContext(ctx, args...)
}

type QueryBuilder struct {
    query  strings.Builder
    args   []interface{}
    params map[string]interface{}
}

func NewQueryBuilder() *QueryBuilder {
    return &QueryBuilder{
        params: make(map[string]interface{}),
    }
}

func (qb *QueryBuilder) AddParam(name string, value interface{}) {
    qb.params[name] = value
}

func (qb *QueryBuilder) Build(baseQuery string) (string, []interface{}) {
    paramCount := 1
    query := baseQuery

    for name, value := range qb.params {
        placeholder := fmt.Sprintf("@%s", name)
        if strings.Contains(query, placeholder) {
            query = strings.ReplaceAll(query, placeholder, fmt.Sprintf("$%d", paramCount))
            qb.args = append(qb.args, value)
            paramCount++
        }
    }

    return query, qb.args
}

type TxFn func(*sql.Tx) error

func (db *DB) WithTransaction(ctx context.Context, fn TxFn) error {
    tx, err := db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }

    defer func() {
        if p := recover(); p != nil {
            tx.Rollback()
            panic(p)
        }
    }()

    if err := fn(tx); err != nil {
        tx.Rollback()
        return err
    }

    return tx.Commit()
}

func SafeOrderBy(column string, validColumns []string) string {
    column = strings.TrimSpace(strings.ToLower(column))
    for _, valid := range validColumns {
        if column == strings.ToLower(valid) {
            return valid
        }
    }
    return validColumns[0] // Default to first valid column
}

func SafeLimit(limit int) int {
    if limit <= 0 {
        return 10 // Default limit
    }
    if limit > 100 {
        return 100 // Maximum limit
    }
    return limit
}

func SafeOffset(offset int) int {
    if offset < 0 {
        return 0
    }
    return offset
}