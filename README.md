# WOLFAI - AI-Powered Investment Platform

Https://wolf-ai.xyz

WOLFAI is an advanced investment platform that leverages artificial intelligence and machine learning to provide portfolio management, real-time market analysis, and automated trading signals.

## Key Features

- **AI-Powered Market Analysis**: LSTM neural networks process real-time market data to predict price movements and trends
- **Portfolio Optimization**: Advanced algorithms optimize portfolio allocation based on risk tolerance and market conditions
- **Risk Management**: Real-time monitoring of portfolio risk metrics with automated alerts
- **Real-Time Trading Signals**: ML-based signals for entry and exit points
- **Automated Retraining**: Self-improving models that adapt to changing market conditions

## Technology Stack

- **Backend**: Go (Golang)
- **Machine Learning**: Python, TensorFlow, scikit-learn
- **Database**: PostgreSQL
- **Caching**: Redis
- **API**: RESTful with JWT authentication

## Prerequisites

- Go 1.21+
- Python 3.9+
- PostgreSQL 14+
- Redis 6+
- Make (optional, for using Makefile commands)

## Quick Start

1. Clone the repository:
```bash
git clone https://github.com/Cryptoprojectsfun/WOLFAI.git
cd WOLFAI
```

2. Install Go dependencies:
```bash
go mod download
```

3. Install Python dependencies:
```bash
pip install -r requirements.txt
```

4. Configure environment:
```bash
cp config/app.example.yaml config/app.yaml
# Edit config/app.yaml with your settings
```

5. Setup databases:
```bash
# Start PostgreSQL and create database
createdb wolfai

# Run migrations
make migrate

# Start Redis server
redis-server
```

6. Start the server:
```bash
make run
```

The server will be available at `http://localhost:8080`

## Architecture

### Components

- **Market Data Pipeline**: Real-time collection and processing of market data
- **ML Service**: LSTM model training and inference
- **Portfolio Service**: Portfolio management and optimization
- **Risk Service**: Risk analysis and monitoring
- **API Layer**: RESTful endpoints with authentication and rate limiting

### Data Flow

1. Market data is collected in real-time through the data pipeline
2. Data is processed and stored in PostgreSQL/Redis
3. ML models analyze data to generate predictions
4. Portfolio and risk services use predictions to optimize allocations
5. Results are exposed through the API layer

## API Documentation

Once the server is running, API documentation is available at:
- Swagger UI: `http://localhost:8080/api/docs`
- OpenAPI JSON: `http://localhost:8080/api/docs.json`

Key endpoints:
- `/api/v1/auth/*`: Authentication endpoints
- `/api/v1/portfolio/*`: Portfolio management
- `/api/v1/analytics/*`: Market analysis and predictions
- `/api/v1/risk/*`: Risk metrics and alerts

## Development

Run tests:
```bash
make test
```

Run linting:
```bash
make lint
```

Generate API documentation:
```bash
make docs
```

## ML Model Training

Train new model:
```bash
python ml/train.py --config config/ml/lstm_config.json --output models/lstm
```

Model configs are in `config/ml/`:
- `lstm_config.json`: LSTM model hyperparameters
- `training_config.json`: Training parameters

## Monitoring

The platform provides several monitoring endpoints:
- `/metrics`: Prometheus metrics
- `/health`: Health check status
- `/status`: System status and version

## Contributing

1. Fork the repository
2. Create feature branch (`git checkout -b feature/name`)
3. Commit changes (`git commit -am 'Add feature'`)
4. Push branch (`git push origin feature/name`)
5. Create Pull Request

## License

MIT License - see LICENSE file for details
