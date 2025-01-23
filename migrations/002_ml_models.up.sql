CREATE TABLE ml_models (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    version VARCHAR(50) NOT NULL,
    type VARCHAR(50) NOT NULL,
    config JSONB NOT NULL,
    metrics JSONB,
    path VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (name, version)
);

CREATE TABLE model_predictions (
    id BIGSERIAL PRIMARY KEY,
    model_id BIGINT REFERENCES ml_models(id),
    symbol VARCHAR(20) NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    predictions JSONB NOT NULL,
    confidence FLOAT NOT NULL,
    features JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (model_id, symbol, timestamp)
);

CREATE TABLE training_jobs (
    id BIGSERIAL PRIMARY KEY,
    model_id BIGINT REFERENCES ml_models(id),
    status VARCHAR(50) NOT NULL,
    config JSONB NOT NULL,
    metrics JSONB,
    logs TEXT,
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_predictions_symbol_timestamp ON model_predictions(symbol, timestamp);
CREATE INDEX idx_model_status ON ml_models(status);
CREATE INDEX idx_training_status ON training_jobs(status);