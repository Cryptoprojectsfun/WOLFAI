-- Drop triggers
DROP TRIGGER IF EXISTS refresh_portfolio_summary_on_portfolio_change ON portfolios;
DROP TRIGGER IF EXISTS refresh_portfolio_summary_on_asset_change ON assets;

-- Drop functions
DROP FUNCTION IF EXISTS refresh_portfolio_summary();

-- Drop materialized view and its indexes
DROP MATERIALIZED VIEW IF EXISTS portfolio_summary;

-- Drop indexes
DROP INDEX IF EXISTS idx_users_email;
DROP INDEX IF EXISTS idx_users_subscription_tier;
DROP INDEX IF EXISTS idx_portfolio_performance_portfolio_id;
DROP INDEX IF EXISTS idx_portfolio_performance_updated_at;
DROP INDEX IF EXISTS idx_assets_symbol;
DROP INDEX IF EXISTS idx_assets_type;
DROP INDEX IF EXISTS idx_assets_last_update;
DROP INDEX IF EXISTS idx_assets_portfolio_symbol;
DROP INDEX IF EXISTS idx_asset_prices_symbol_price;
DROP INDEX IF EXISTS idx_market_analysis_sentiment;
DROP INDEX IF EXISTS idx_predictions_timeframe;
DROP INDEX IF EXISTS idx_trading_signals_created_at;
DROP INDEX IF EXISTS idx_active_predictions;
DROP INDEX IF EXISTS idx_asset_prices_time_range;

-- Remove constraints
ALTER TABLE assets DROP CONSTRAINT IF EXISTS check_positive_quantity;
ALTER TABLE asset_prices DROP CONSTRAINT IF EXISTS check_positive_price;
ALTER TABLE predictions DROP CONSTRAINT IF EXISTS check_valid_timeframe;
ALTER TABLE market_analysis DROP CONSTRAINT IF EXISTS check_valid_sentiment;