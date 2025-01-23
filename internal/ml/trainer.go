package ml

import (
    "context"
    "database/sql"
    "encoding/json"
    "fmt"
    "time"
)

type ModelTrainer struct {
    db        *sql.DB
    manager   *ModelManager
    service   *Service
}

type TrainingSchedule struct {
    ModelName     string          `json:"model_name"`
    Version       string          `json:"version"`
    Interval      time.Duration   `json:"interval"`
    DataWindow    time.Duration   `json:"data_window"`
    MinSamples    int            `json:"min_samples"`
    Config        json.RawMessage `json:"config"`
}

func NewModelTrainer(db *sql.DB, manager *ModelManager, service *Service) *ModelTrainer {
    return &ModelTrainer{
        db:      db,
        manager: manager,
        service: service,
    }
}

func (t *ModelTrainer) StartScheduledTraining(ctx context.Context, schedule *TrainingSchedule) error {
    ticker := time.NewTicker(schedule.Interval)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
            
        case <-ticker.C:
            if err := t.runTrainingCycle(ctx, schedule); err != nil {
                fmt.Printf("Training cycle failed: %v\n", err)
            }
        }
    }
}

func (t *ModelTrainer) runTrainingCycle(ctx context.Context, schedule *TrainingSchedule) error {
    // Check if we have enough data
    dataCount, err := t.checkDataAvailability(ctx, schedule.DataWindow)
    if err != nil {
        return err
    }

    if dataCount < schedule.MinSamples {
        return fmt.Errorf("insufficient data: got %d, need %d", dataCount, schedule.MinSamples)
    }

    // Create new version for retrained model
    newVersion := fmt.Sprintf("%s.%d", schedule.Version, time.Now().Unix())

    // Start training job
    config := &TrainingConfig{
        ModelName:   schedule.ModelName,
        Version:    newVersion,
        DataConfig: json.RawMessage(`{
            "window": "` + schedule.DataWindow.String() + `"
        }`),
        ModelConfig: schedule.Config,
    }

    jobID, err := t.service.StartTraining(ctx, config)
    if err != nil {
        return err
    }

    // Monitor training progress
    return t.monitorTraining(ctx, jobID, schedule.ModelName, newVersion)
}

func (t *ModelTrainer) checkDataAvailability(ctx context.Context, window time.Duration) (int, error) {
    query := `
        SELECT COUNT(*) 
        FROM market_data 
        WHERE timestamp >= $1
    `
    
    var count int
    err := t.db.QueryRowContext(ctx, query, time.Now().Add(-window)).Scan(&count)
    return count, err
}

func (t *ModelTrainer) monitorTraining(ctx context.Context, jobID int64, modelName, version string) error {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()

    timeout := time.After(24 * time.Hour)

    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
            
        case <-timeout:
            return fmt.Errorf("training timeout")
            
        case <-ticker.C:
            status, err := t.service.GetTrainingStatus(ctx, jobID)
            if err != nil {
                return err
            }

            switch status {
            case "completed":
                // Activate new model version
                return t.activateNewModel(ctx, modelName, version)
                
            case "failed":
                return fmt.Errorf("training failed")
            }
        }
    }
}

func (t *ModelTrainer) activateNewModel(ctx context.Context, modelName, version string) error {
    // Archive current active model
    models, err := t.manager.ListModels(ctx, "active")
    if err != nil {
        return err
    }

    for _, model := range models {
        if model.Name == modelName {
            if err := t.manager.ArchiveModel(ctx, model.Name, model.Version); err != nil {
                return err
            }
        }
    }

    // Activate new model
    return t.manager.UpdateModelStatus(ctx, modelName, version, "active")
}
