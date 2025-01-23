package market

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type MarketDataCollector struct {
	db        *sql.DB
	apiKey    string
	provider  string
	symbols   []string
	interval  time.Duration
	stopChan  chan struct{}
}

func NewMarketDataCollector(
	db *sql.DB,
	provider string,
	apiKey string,
	symbols []string,
	interval time.Duration,
) *MarketDataCollector {
	return &MarketDataCollector{
		db:       db,
		provider: provider,
		apiKey:   apiKey,
		symbols:  symbols,
		interval: interval,
		stopChan: make(chan struct{}),
	}
}

func (c *MarketDataCollector) Start(ctx context.Context) error {
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-c.stopChan:
			return nil
		case <-ticker.C:
			if err := c.collect(ctx); err != nil {
				fmt.Printf("Error collecting market data: %v\n", err)
			}
		}
	}
}

func (c *MarketDataCollector) Stop() {
	close(c.stopChan)
}

func (c *MarketDataCollector) collect(ctx context.Context) error {
	for _, symbol := range c.symbols {
		data, err := c.fetchMarketData(ctx, symbol)
		if err != nil {
			return fmt.Errorf("failed to fetch data for %s: %v", symbol, err)
		}

		if err := c.saveMarketData(ctx, symbol, data); err != nil {
			return fmt.Errorf("failed to save data for %s: %v", symbol, err)
		}
	}
	return nil
}

func (c *MarketDataCollector) fetchMarketData(ctx context.Context, symbol string) (map[string]interface{}, error) {
	url := fmt.Sprintf("https://api.%s.com/v1/data/%s?apikey=%s", c.provider, symbol, c.apiKey)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	return data, nil
}

func (c *MarketDataCollector) saveMarketData(ctx context.Context, symbol string, data map[string]interface{}) error {
	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `
		INSERT INTO market_data (symbol, timestamp, open, high, low, close, volume)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (symbol, timestamp) DO UPDATE 
		SET open = EXCLUDED.open,
			high = EXCLUDED.high,
			low = EXCLUDED.low,
			close = EXCLUDED.close,
			volume = EXCLUDED.volume
	`

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, candle := range data["candles"].([]interface{}) {
		c := candle.(map[string]interface{})
		_, err = stmt.ExecContext(ctx,
			symbol,
			c["timestamp"],
			c["open"],
			c["high"],
			c["low"],
			c["close"],
			c["volume"],
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}
