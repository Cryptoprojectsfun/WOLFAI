-- Create users table
CREATE TABLE users (
    id UUID PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    hashed_password VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'user',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL,
    last_login_at TIMESTAMP WITH TIME ZONE,
    subscription_tier VARCHAR(50) NOT NULL DEFAULT 'free'
);

-- Create portfolios table
CREATE TABLE portfolios (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    total_value DECIMAL(20, 8) NOT NULL DEFAULT 0,
    risk_score DECIMAL(5, 2),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL,
    CONSTRAINT unique_portfolio_name_per_user UNIQUE (user_id, name)
);

-- Create assets table
CREATE TABLE assets (
    id UUID PRIMARY KEY,
    portfolio_id UUID NOT NULL REFERENCES portfolios(id) ON DELETE CASCADE,
    symbol VARCHAR(50) NOT NULL,
    type VARCHAR(50) NOT NULL,
    quantity DECIMAL(20, 8) NOT NULL,
    value DECIMAL(20, 8) NOT NULL,
    last_update TIMESTAMP WITH TIME ZONE NOT NULL
);

-- Create portfolio_performance table
CREATE TABLE portfolio_performance (
    id UUID PRIMARY KEY,
    portfolio_id UUID NOT NULL REFERENCES portfolios(id) ON DELETE CASCADE,
    daily_return DECIMAL(10, 4),
    weekly_return DECIMAL(10, 4),
    monthly_return DECIMAL(10, 4),
    yearly_return DECIMAL(10, 4),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL
);

-- Create asset_prices table
CREATE TABLE asset_prices (
    id UUID PRIMARY KEY,
    symbol VARCHAR(50) NOT NULL,
    price DECIMAL(20, 8) NOT NULL,
    volume DECIMAL(20, 8) NOT NULL,
    date TIMESTAMP WITH TIME ZONE NOT NULL,
    CONSTRAINT unique_price_per_symbol_date UNIQUE (symbol, date)
);

-- Create market_analysis table
CREATE TABLE market_analysis (
    id UUID PRIMARY KEY,
    asset_symbol VARCHAR(50) NOT NULL,
    sentiment DECIMAL(5, 2) NOT NULL,
    volume_24h DECIMAL(20, 8) NOT NULL,
    price_change_24h DECIMAL(10, 4) NOT NULL,
    trend_strength DECIMAL(5, 2) NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL
);

-- Create predictions table
CREATE TABLE predictions (
    id UUID PRIMARY KEY,
    asset_symbol VARCHAR(50) NOT NULL,
    timeframe VARCHAR(10) NOT NULL,
    predicted_high DECIMAL(20, 8) NOT NULL,
    predicted_low DECIMAL(20, 8) NOT NULL,
    confidence DECIMAL(5, 2) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    valid_until TIMESTAMP WITH TIME ZONE NOT NULL
);

-- Create indicators table
CREATE TABLE indicators (
    id UUID PRIMARY KEY,
    prediction_id UUID NOT NULL REFERENCES predictions(id) ON DELETE CASCADE,
    name VARCHAR(50) NOT NULL,
    value DECIMAL(20, 8) NOT NULL,
    weight DECIMAL(5, 2) NOT NULL
);

-- Create trading_signals table
CREATE TABLE trading_signals (
    id UUID PRIMARY KEY,
    analysis_id UUID NOT NULL REFERENCES market_analysis(id) ON DELETE CASCADE,
    symbol VARCHAR(50) NOT NULL,
    type VARCHAR(50) NOT NULL,
    strength DECIMAL(5, 2) NOT NULL,
    description TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL
);

-- Create indices
CREATE INDEX idx_portfolios_user_id ON portfolios(user_id);
CREATE INDEX idx_assets_portfolio_id ON assets(portfolio_id);
CREATE INDEX idx_asset_prices_symbol_date ON asset_prices(symbol, date);
CREATE INDEX idx_market_analysis_symbol ON market_analysis(asset_symbol);
CREATE INDEX idx_predictions_symbol ON predictions(asset_symbol);
CREATE INDEX idx_trading_signals_symbol ON trading_signals(symbol);

-- Create extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "btree_gist";