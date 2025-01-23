package pipeline

import (
    "context"
    "encoding/json"
    "fmt"
    "sync"
    "time"

    "github.com/go-redis/redis/v8"
    "github.com/Cryptoprojectsfun/quantai-clone/internal/cache"
    "github.com/Cryptoprojectsfun/quantai-clone/internal/models"
    "github.com/Cryptoprojectsfun/quantai-clone/internal/services/market"
)

type MarketDataPipeline struct {
    collector  *market.MarketDataCollector
    cache      *cache.MarketDataCache
    rdb        *redis.Client
    pubsub     *redis.PubSub
    batchSize  int
    interval   time.Duration
    symbols    []string
    updateChan chan struct{}
    mu         sync.RWMutex
}

func NewMarketDataPipeline(
    collector *market.MarketDataCollector,
    cache *cache.MarketDataCache,
    rdb *redis.Client,
    batchSize int,
    interval time.Duration,
) *MarketDataPipeline {
    return &MarketDataPipeline{
        collector:  collector,
        cache:      cache,
        rdb:        rdb,
        batchSize:  batchSize,
        interval:   interval,
        updateChan: make(chan struct{}, 1),
    }
}

func (p *MarketDataPipeline) Start(ctx context.Context) error {
    // Subscribe to symbol updates
    p.pubsub = p.rdb.Subscribe(ctx, "market:symbols:update")
    go p.handleSymbolUpdates(ctx)

    // Start data collection
    ticker := time.NewTicker(p.interval)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-ticker.C:
            if err := p.collectAndProcess(ctx); err != nil {
                fmt.Printf("Pipeline error: %v\n", err)
            }
        case <-p.updateChan:
            if err := p.collectAndProcess(ctx); err != nil {
                fmt.Printf("Pipeline error after update: %v\n", err)
            }
        }
    }
}

func (p *MarketDataPipeline) handleSymbolUpdates(ctx context.Context) {
    ch := p.pubsub.Channel()
    for {
        select {
        case <-ctx.Done():
            return
        case msg := <-ch:
            var symbols []string
            if err := json.Unmarshal([]byte(msg.Payload), &symbols); err != nil {
                fmt.Printf("Failed to unmarshal symbols update: %v\n", err)
                continue
            }
            p.updateSymbols(symbols)
            select {
            case p.updateChan <- struct{}{}:
            default:
            }
        }
    }
}

func (p *MarketDataPipeline) updateSymbols(symbols []string) {
    p.mu.Lock()
    p.symbols = symbols
    p.mu.Unlock()
}

func (p *MarketDataPipeline) collectAndProcess(ctx context.Context) error {
    p.mu.RLock()
    symbols := p.symbols
    p.mu.RUnlock()

    if len(symbols) == 0 {
        return nil
    }

    // Collect data in batches
    for i := 0; i < len(symbols); i += p.batchSize {
        end := i + p.batchSize
        if end > len(symbols) {
            end = len(symbols)
        }
        batch := symbols[i:end]

        // Collect market data
        data, err := p.collector.CollectBatch(ctx, batch)
        if err != nil {
            return fmt.Errorf("failed to collect data: %v", err)
        }

        // Process and cache data
        if err := p.processData(ctx, data); err != nil {
            return fmt.Errorf("failed to process data: %v", err)
        }

        // Notify subscribers
        if err := p.notifyUpdates(ctx, batch); err != nil {
            return fmt.Errorf("failed to notify updates: %v", err)
        }
    }

    return nil
}

func (p *MarketDataPipeline) processData(ctx context.Context, data map[string]models.MarketData) error {
    // Update cache
    for symbol, marketData := range data {
        if err := p.cache.SetMarketData(ctx, symbol, &marketData); err != nil {
            return err
        }
    }

    // Store in Redis for real-time access
    pipe := p.rdb.Pipeline()
    for symbol, marketData := range data {
        key := fmt.Sprintf("market:realtime:%s", symbol)
        jsonData, err := json.Marshal(marketData)
        if err != nil {
            return err
        }
        pipe.Set(ctx, key, jsonData, time.Hour)
    }
    _, err := pipe.Exec(ctx)
    return err
}

func (p *MarketDataPipeline) notifyUpdates(ctx context.Context, symbols []string) error {
    // Publish updates to subscribers
    for _, symbol := range symbols {
        channel := fmt.Sprintf("market:updates:%s", symbol)
        if err := p.rdb.Publish(ctx, channel, "updated").Err(); err != nil {
            return err
        }
    }
    return nil
}
