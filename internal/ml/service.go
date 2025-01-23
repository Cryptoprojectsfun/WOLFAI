package ml

import (
    "context"
    "database/sql"
    "encoding/json"
    "fmt"
    "os"
    "os/exec"
    "path/filepath"
    "time"
)

type Service struct {
    db        *sql.DB
    modelPath string
}

type PredictionRequest struct {
    Symbol    string          `json:"symbol"`
    Features  []float64       `json:"features"`
    ModelName string          `json:"model_name"`
    Version   string          `json:"version"`
}

type PredictionResponse struct {
    Symbol      string    `json:"symbol"`
    Timestamp   time.Time `json:"timestamp"`
    Predictions struct {
        PriceHigh   float64 `json:"price_high"`
        PriceLow    float64 `json:"price_low"`
        PriceClose  float64 `json:"price_close"`
        Direction   float64 `json:"direction"`
    } `json:"predictions"`
    Confidence  float64 `json:"confidence"`
}

type TrainingConfig struct {
    ModelName    string          `json:"model_name"`
    Version      string          `json:"version"`
    DataConfig   json.RawMessage `json:"data_config"`
    ModelConfig  json.RawMessage `json:"model_config"`
    TrainConfig  json.RawMessage `json:"train_config"`
}

func NewService(db *sql.DB, modelPath string) *Service {
    return &Service{
        db:        db,
        modelPath: modelPath,
    }
}

func (s *Service) Predict(ctx context.Context, req *PredictionRequest) (*PredictionResponse, error) {
    // Get latest model version if not specified
    if req.Version == "" {
        query := `
            SELECT version FROM ml_models 
            WHERE name = $1 AND status = 'active'
            ORDER BY created_at DESC LIMIT 1
        `
        err := s.db.QueryRowContext(ctx, query, req.ModelName).Scan(&req.Version)
        if err != nil {
            return nil, fmt.Errorf("failed to get latest model version: %v", err)
        }
    }

    // Prepare input data
    inputData := map[string]interface{}{
        "features": req.Features,
    }
    inputJSON, err := json.Marshal(inputData)
    if err != nil {
        return nil, err
    }

    // Call Python prediction script
    scriptPath := filepath.Join(s.modelPath, "predict.py")
    modelPath := filepath.Join(s.modelPath, req.ModelName, req.Version)

    cmd := exec.CommandContext(ctx, "python", scriptPath, 
        "--model-path", modelPath,
        "--input", string(inputJSON),
    )

    output, err := cmd.Output()
    if err != nil {
        return nil, fmt.Errorf("prediction failed: %v", err)
    }

    var resp PredictionResponse
    if err := json.Unmarshal(output, &resp); err != nil {
        return nil, err
    }

    resp.Symbol = req.Symbol
    resp.Timestamp = time.Now()

    // Save prediction to database
    query := `
        INSERT INTO model_predictions (
            model_id, symbol, timestamp, predictions, confidence, features
        ) VALUES (
            (SELECT id FROM ml_models WHERE name = $1 AND version = $2),
            $3, $4, $5, $6, $7
        )
    `
    predictions, _ := json.Marshal(resp.Predictions)
    features, _ := json.Marshal(req.Features)

    _, err = s.db.ExecContext(ctx, query,
        req.ModelName, req.Version,
        resp.Symbol, resp.Timestamp,
        predictions, resp.Confidence,
        features,
    )
    if err != nil {
        return nil, fmt.Errorf("failed to save prediction: %v", err)
    }

    return &resp, nil
}

func (s *Service) StartTraining(ctx context.Context, config *TrainingConfig) (int64, error) {
    // Create training job record
    tx, err := s.db.BeginTx(ctx, nil)
    if err != nil {
        return 0, err
    }
    defer tx.Rollback()

    var jobID int64
    query := `
        INSERT INTO training_jobs (
            model_id, status, config, started_at
        ) VALUES (
            (SELECT id FROM ml_models WHERE name = $1 AND version = $2),
            'pending', $3, $4
        ) RETURNING id
    `
    configJSON, _ := json.Marshal(config)
    err = tx.QueryRowContext(ctx, query,
        config.ModelName, config.Version,
        configJSON, time.Now(),
    ).Scan(&jobID)
    if err != nil {
        return 0, fmt.Errorf("failed to create training job: %v", err)
    }

    // Prepare training data and start training process
    go func() {
        if err := s.runTraining(config, jobID); err != nil {
            s.updateTrainingStatus(jobID, "failed", err.Error())
            return
        }
    }()

    if err := tx.Commit(); err != nil {
        return 0, err
    }

    return jobID, nil
}

func (s *Service) runTraining(config *TrainingConfig, jobID int64) error {
    scriptPath := filepath.Join(s.modelPath, "train.py")
    modelPath := filepath.Join(s.modelPath, config.ModelName, config.Version)

    if err := os.MkdirAll(modelPath, 0755); err != nil {
        return fmt.Errorf("failed to create model directory: %v", err)
    }

    configJSON, _ := json.Marshal(config)
    cmd := exec.Command("python", scriptPath,
        "--config", string(configJSON),
        "--output", modelPath,
    )

    output, err := cmd.CombinedOutput()
    if err != nil {
        return fmt.Errorf("training failed: %v, output: %s", err, string(output))
    }

    return s.updateTrainingStatus(jobID, "completed", string(output))
}

func (s *Service) updateTrainingStatus(jobID int64, status, logs string) error {
    query := `
        UPDATE training_jobs 
        SET status = $1, logs = $2, completed_at = $3, updated_at = $3
        WHERE id = $4
    `
    _, err := s.db.Exec(query, status, logs, time.Now(), jobID)
    return err
}

func (s *Service) GetTrainingStatus(ctx context.Context, jobID int64) (string, error) {
    var status string
    query := "SELECT status FROM training_jobs WHERE id = $1"
    err := s.db.QueryRowContext(ctx, query, jobID).Scan(&status)
    return status, err
}
