package cache

import (
    "context"
    "encoding/json"
    "fmt"
    "time"

    "github.com/go-redis/redis/v8"
    "github.com/Cryptoprojectsfun/quantai-clone/internal/models"
)

type MarketDataCache struct {
    client *redis.Client
    ttl    time.Duration
}

func NewMarketDataCache(client *redis.Client, ttl time.Duration) *MarketDataCache {
    return &MarketDataCache{
        client: client,
        ttl:    ttl,
    }
}

func (c *MarketDataCache) GetMarketData(ctx context.Context, symbol string) (*models.MarketData, error) {
    key := fmt.Sprintf("market:data:%s", symbol)
    data, err := c.client.Get(ctx, key).Bytes()
    if err == redis.Nil {
        return nil, nil
    }
    if err != nil {
        return nil, err
    }

    var marketData models.MarketData
    if err := json.Unmarshal(data, &marketData); err != nil {
        return nil, err
    }

    return &marketData, nil
}

func (c *MarketDataCache) SetMarketData(ctx context.Context, symbol string, data *models.MarketData) error {
    key := fmt.Sprintf("market:data:%s", symbol)
    
    jsonData, err := json.Marshal(data)
    if err != nil {
        return err
    }

    return c.client.Set(ctx, key, jsonData, c.ttl).Err()
}

func (c *MarketDataCache) GetHistoricalData(ctx context.Context, symbol string, start, end time.Time) ([]models.MarketData, error) {
    key := fmt.Sprintf("market:historical:%s:%d:%d", symbol, start.Unix(), end.Unix())
    data, err := c.client.Get(ctx, key).Bytes()
    if err == redis.Nil {
        return nil, nil
    }
    if err != nil {
        return nil, err
    }

    var historicalData []models.MarketData
    if err := json.Unmarshal(data, &historicalData); err != nil {
        return nil, err
    }

    return historicalData, nil
}

func (c *MarketDataCache) SetHistoricalData(ctx context.Context, symbol string, start, end time.Time, data []models.MarketData) error {
    key := fmt.Sprintf("market:historical:%s:%d:%d", symbol, start.Unix(), end.Unix())
    
    jsonData, err := json.Marshal(data)
    if err != nil {
        return err
    }

    return c.client.Set(ctx, key, jsonData, c.ttl).Err()
}

func (c *MarketDataCache) InvalidateSymbol(ctx context.Context, symbol string) error {
    pattern := fmt.Sprintf("market:*:%s:*", symbol)
    
    iter := c.client.Scan(ctx, 0, pattern, 0).Iterator()
    for iter.Next(ctx) {
        if err := c.client.Del(ctx, iter.Val()).Err(); err != nil {
            return err
        }
    }
    
    return iter.Err()
}

func (c *MarketDataCache) GetCachedSymbols(ctx context.Context) ([]string, error) {
    pattern := "market:data:*"
    var symbols []string

    iter := c.client.Scan(ctx, 0, pattern, 0).Iterator()
    for iter.Next(ctx) {
        key := iter.Val()
        symbol := key[len("market:data:"):]
        symbols = append(symbols, symbol)
    }
    
    if err := iter.Err(); err != nil {
        return nil, err
    }

    return symbols, nil
}

func (c *MarketDataCache) PurgeTTLExpired(ctx context.Context) error {
    pattern := "market:*"
    
    iter := c.client.Scan(ctx, 0, pattern, 0).Iterator()
    for iter.Next(ctx) {
        key := iter.Val()
        ttl, err := c.client.TTL(ctx, key).Result()
        if err != nil {
            return err
        }
        
        if ttl.Seconds() <= 0 {
            if err := c.client.Del(ctx, key).Err(); err != nil {
                return err
            }
        }
    }
    
    return iter.Err()
}
