package lstm

import (
    "context"
    "database/sql"
    "encoding/json"
    "fmt"
    "os"
    "os/exec"
    "path/filepath"
    "sync"
    "time"
)

type Service struct {
    db          *sql.DB
    modelPath   string
    modelCache  map[string]*Model
    cacheMutex  sync.RWMutex
    batchSize   int
    maxRetries  int
}

type Model struct {
    Name      string
    Version   string
    Process   *exec.Cmd
    InputChan chan []float64
    OutputChan chan Prediction
    LastUsed   time.Time
}

type Prediction struct {
    PriceHigh  float64
    PriceLow   float64
    PriceClose float64
    Direction  float64
    Confidence float64
}

func NewService(db *sql.DB, modelPath string) *Service {
    return &Service{
        db:         db,
        modelPath:  modelPath,
        modelCache: make(map[string]*Model),
        batchSize:  32,
        maxRetries: 3,
    }
}

func (s *Service) GetModel(name, version string) (*Model, error) {
    s.cacheMutex.RLock()
    modelKey := fmt.Sprintf("%s@%s", name, version)
    model, exists := s.modelCache[modelKey]
    s.cacheMutex.RUnlock()

    if exists {
        model.LastUsed = time.Now()
        return model, nil
    }

    s.cacheMutex.Lock()
    defer s.cacheMutex.Unlock()

    // Check again in case another goroutine loaded it
    if model, exists = s.modelCache[modelKey]; exists {
        model.LastUsed = time.Now()
        return model, nil
    }

    // Load model
    model, err := s.loadModel(name, version)
    if err != nil {
        return nil, err
    }

    s.modelCache[modelKey] = model
    return model, nil
}

func (s *Service) loadModel(name, version string) (*Model, error) {
    modelPath := filepath.Join(s.modelPath, name, version)
    if _, err := os.Stat(modelPath); os.IsNotExist(err) {
        return nil, fmt.Errorf("model not found: %s@%s", name, version)
    }

    // Start Python process for model inference
    cmd := exec.Command("python",
        filepath.Join(s.modelPath, "serve.py"),
        "--model-path", modelPath,
    )

    // Set up pipes for communication
    stdinPipe, err := cmd.StdinPipe()
    if err != nil {
        return nil, err
    }

    stdoutPipe, err := cmd.StdoutPipe()
    if err != nil {
        return nil, err
    }

    if err := cmd.Start(); err != nil {
        return nil, err
    }

    model := &Model{
        Name:       name,
        Version:    version,
        Process:    cmd,
        InputChan:  make(chan []float64, s.batchSize),
        OutputChan: make(chan Prediction, s.batchSize),
        LastUsed:   time.Now(),
    }

    // Start goroutines for handling I/O
    go func() {
        defer stdinPipe.Close()
        encoder := json.NewEncoder(stdinPipe)

        for features := range model.InputChan {
            if err := encoder.Encode(features); err != nil {
                fmt.Printf("Failed to encode features: %v\n", err)
                continue
            }
        }
    }()

    go func() {
        defer close(model.OutputChan)
        decoder := json.NewDecoder(stdoutPipe)

        for {
            var pred Prediction
            if err := decoder.Decode(&pred); err != nil {
                fmt.Printf("Failed to decode prediction: %v\n", err)
                return
            }
            model.OutputChan <- pred
        }
    }()

    return model, nil
}

func (s *Service) Predict(ctx context.Context, name, version string, features []float64) (*Prediction, error) {
    var lastErr error
    for i := 0; i < s.maxRetries; i++ {
        model, err := s.GetModel(name, version)
        if err != nil {
            lastErr = err
            continue
        }

        select {
        case model.InputChan <- features:
        case <-ctx.Done():
            return nil, ctx.Err()
        }

        select {
        case pred := <-model.OutputChan:
            return &pred, nil
        case <-ctx.Done():
            return nil, ctx.Err()
        }
    }

    return nil, fmt.Errorf("prediction failed after %d retries: %v", s.maxRetries, lastErr)
}

func (s *Service) BatchPredict(ctx context.Context, name, version string, featuresBatch [][]float64) ([]Prediction, error) {
    model, err := s.GetModel(name, version)
    if err != nil {
        return nil, err
    }

    predictions := make([]Prediction, 0, len(featuresBatch))
    
    for i := 0; i < len(featuresBatch); i += s.batchSize {
        end := i + s.batchSize
        if end > len(featuresBatch) {
            end = len(featuresBatch)
        }
        batch := featuresBatch[i:end]

        // Send batch
        for _, features := range batch {
            select {
            case model.InputChan <- features:
            case <-ctx.Done():
                return nil, ctx.Err()
            }
        }

        // Receive predictions
        for range batch {
            select {
            case pred := <-model.OutputChan:
                predictions = append(predictions, pred)
            case <-ctx.Done():
                return nil, ctx.Err()
            }
        }
    }

    return predictions, nil
}

func (s *Service) CleanupCache(maxAge time.Duration) {
    s.cacheMutex.Lock()
    defer s.cacheMutex.Unlock()

    now := time.Now()
    for key, model := range s.modelCache {
        if now.Sub(model.LastUsed) > maxAge {
            model.Process.Process.Kill()
            delete(s.modelCache, key)
        }
    }
}
