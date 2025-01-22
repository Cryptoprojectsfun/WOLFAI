-- Additional indexes for performance optimization

-- Users table indexes
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_subscription_tier ON users(subscription_tier);

-- Portfolio performance indexes
CREATE INDEX idx_portfolio_performance_portfolio_id ON portfolio_performance(portfolio_id);
CREATE INDEX idx_portfolio_performance_updated_at ON portfolio_performance(updated_at);

-- Asset specific indexes
CREATE INDEX idx_assets_symbol ON assets(symbol);
CREATE INDEX idx_assets_type ON assets(type);
CREATE INDEX idx_assets_last_update ON assets(last_update);

-- Composite indexes for common queries
CREATE INDEX idx_assets_portfolio_symbol ON assets(portfolio_id, symbol);
CREATE INDEX idx_asset_prices_symbol_price ON asset_prices(symbol, price);
CREATE INDEX idx_market_analysis_sentiment ON market_analysis(asset_symbol, sentiment);

-- Time-based indexes for historical queries
CREATE INDEX idx_predictions_timeframe ON predictions(timeframe, created_at);
CREATE INDEX idx_trading_signals_created_at ON trading_signals(created_at);

-- Create materialized view for common portfolio calculations
CREATE MATERIALIZED VIEW portfolio_summary AS
SELECT 
    p.id AS portfolio_id,
    p.user_id,
    p.name,
    COUNT(a.id) AS asset_count,
    SUM(a.value) AS total_value,
    MAX(pp.daily_return) AS latest_daily_return,
    MAX(pp.monthly_return) AS latest_monthly_return,
    MAX(pp.yearly_return) AS latest_yearly_return
FROM portfolios p
LEFT JOIN assets a ON p.id = a.portfolio_id
LEFT JOIN portfolio_performance pp ON p.id = pp.portfolio_id
GROUP BY p.id, p.user_id, p.name;

-- Create index on materialized view
CREATE UNIQUE INDEX idx_portfolio_summary_id ON portfolio_summary(portfolio_id);
CREATE INDEX idx_portfolio_summary_user ON portfolio_summary(user_id);

-- Add function for refreshing materialized view
CREATE OR REPLACE FUNCTION refresh_portfolio_summary()
RETURNS trigger AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY portfolio_summary;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- Create triggers to keep materialized view updated
CREATE TRIGGER refresh_portfolio_summary_on_portfolio_change
    AFTER INSERT OR UPDATE OR DELETE ON portfolios
    FOR EACH STATEMENT
    EXECUTE FUNCTION refresh_portfolio_summary();

CREATE TRIGGER refresh_portfolio_summary_on_asset_change
    AFTER INSERT OR UPDATE OR DELETE ON assets
    FOR EACH STATEMENT
    EXECUTE FUNCTION refresh_portfolio_summary();

-- Create partial indexes for active predictions
CREATE INDEX idx_active_predictions ON predictions(asset_symbol, valid_until)
WHERE valid_until > NOW();

-- Create GiST index for time range queries
CREATE INDEX idx_asset_prices_time_range ON asset_prices USING GIST (
    symbol,
    daterange(date, date, '[]')
);

-- Add constraints for data validation
ALTER TABLE assets
ADD CONSTRAINT check_positive_quantity
CHECK (quantity > 0);

ALTER TABLE asset_prices
ADD CONSTRAINT check_positive_price
CHECK (price > 0);

ALTER TABLE predictions
ADD CONSTRAINT check_valid_timeframe
CHECK (timeframe IN ('1h', '4h', '24h', '7d'));

ALTER TABLE market_analysis
ADD CONSTRAINT check_valid_sentiment
CHECK (sentiment >= -1 AND sentiment <= 1);

-- Create hypertable for time-series data (requires TimescaleDB extension)
-- Uncomment if using TimescaleDB
-- SELECT create_hypertable('asset_prices', 'date');
-- SELECT create_hypertable('trading_signals', 'created_at');