package ml

import (
    "context"
    "database/sql"
    "encoding/json"
    "fmt"
    "time"
)

type ModelManager struct {
    db *sql.DB
}

type ModelInfo struct {
    ID        int64           `json:"id"`
    Name      string         `json:"name"`
    Version   string         `json:"version"`
    Type      string         `json:"type"`
    Config    json.RawMessage `json:"config"`
    Status    string         `json:"status"`
    Metrics   json.RawMessage `json:"metrics,omitempty"`
    CreatedAt time.Time      `json:"created_at"`
    UpdatedAt time.Time      `json:"updated_at"`
}

func NewModelManager(db *sql.DB) *ModelManager {
    return &ModelManager{db: db}
}

func (m *ModelManager) RegisterModel(ctx context.Context, info ModelInfo) error {
    query := `
        INSERT INTO ml_models (
            name, version, type, config, status, created_at, updated_at
        ) VALUES (
            $1, $2, $3, $4, $5, $6, $6
        )
    `
    
    _, err := m.db.ExecContext(ctx, query,
        info.Name,
        info.Version,
        info.Type,
        info.Config,
        info.Status,
        time.Now(),
    )
    return err
}

func (m *ModelManager) UpdateModelStatus(ctx context.Context, name, version, status string) error {
    query := `
        UPDATE ml_models 
        SET status = $1, updated_at = $2
        WHERE name = $3 AND version = $4
    `
    
    result, err := m.db.ExecContext(ctx, query, status, time.Now(), name, version)
    if err != nil {
        return err
    }

    rows, err := result.RowsAffected()
    if err != nil {
        return err
    }

    if rows == 0 {
        return fmt.Errorf("model not found: %s@%s", name, version)
    }

    return nil
}

func (m *ModelManager) UpdateModelMetrics(ctx context.Context, name, version string, metrics json.RawMessage) error {
    query := `
        UPDATE ml_models 
        SET metrics = $1, updated_at = $2
        WHERE name = $3 AND version = $4
    `
    
    _, err := m.db.ExecContext(ctx, query, metrics, time.Now(), name, version)
    return err
}

func (m *ModelManager) GetModel(ctx context.Context, name, version string) (*ModelInfo, error) {
    query := `
        SELECT id, name, version, type, config, status, metrics, created_at, updated_at
        FROM ml_models
        WHERE name = $1 AND version = $2
    `

    var info ModelInfo
    err := m.db.QueryRowContext(ctx, query, name, version).Scan(
        &info.ID,
        &info.Name,
        &info.Version,
        &info.Type,
        &info.Config,
        &info.Status,
        &info.Metrics,
        &info.CreatedAt,
        &info.UpdatedAt,
    )
    if err != nil {
        return nil, err
    }

    return &info, nil
}

func (m *ModelManager) ListModels(ctx context.Context, status string) ([]ModelInfo, error) {
    var query string
    var args []interface{}

    if status != "" {
        query = `
            SELECT id, name, version, type, config, status, metrics, created_at, updated_at
            FROM ml_models
            WHERE status = $1
            ORDER BY created_at DESC
        `
        args = []interface{}{status}
    } else {
        query = `
            SELECT id, name, version, type, config, status, metrics, created_at, updated_at
            FROM ml_models
            ORDER BY created_at DESC
        `
    }

    rows, err := m.db.QueryContext(ctx, query, args...)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var models []ModelInfo
    for rows.Next() {
        var info ModelInfo
        err := rows.Scan(
            &info.ID,
            &info.Name,
            &info.Version,
            &info.Type,
            &info.Config,
            &info.Status,
            &info.Metrics,
            &info.CreatedAt,
            &info.UpdatedAt,
        )
        if err != nil {
            return nil, err
        }
        models = append(models, info)
    }

    return models, nil
}

func (m *ModelManager) ArchiveModel(ctx context.Context, name, version string) error {
    query := `
        UPDATE ml_models 
        SET status = 'archived', updated_at = $1
        WHERE name = $2 AND version = $3 AND status != 'archived'
    `
    
    result, err := m.db.ExecContext(ctx, query, time.Now(), name, version)
    if err != nil {
        return err
    }

    rows, err := result.RowsAffected()
    if err != nil {
        return err
    }

    if rows == 0 {
        return fmt.Errorf("model not found or already archived: %s@%s", name, version)
    }

    return nil
}
