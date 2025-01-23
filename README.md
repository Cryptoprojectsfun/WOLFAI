# WOLFAI - AI-Powered Investment Platform

WOLFAI is an advanced investment platform using AI for portfolio management and market analysis.

## Project Structure

```
.
├── cmd/                    # Application entry points
│   └── server/            # Main server application
├── config/                # Configuration files
│   ├── app.yaml           # Application config
│   └── ml/                # ML model configs
├── internal/              # Private application code
│   ├── api/               # API handlers and routes
│   ├── auth/              # Authentication logic
│   ├── ml/                # Machine learning models
│   ├── models/            # Data models
│   └── services/          # Business logic
├── migrations/            # Database migrations
├── pkg/                   # Public packages
│   ├── database/          # Database utilities
│   └── logger/            # Logging utilities
└── scripts/               # Utility scripts
```

## Features

- Real-time market analysis
- Portfolio optimization
- Trading signals
- Risk management
- User authentication
- Admin dashboard

## Setup

1. Clone and install dependencies:
```bash
git clone https://github.com/Cryptoprojectsfun/quantai-clone.git
cd quantai-clone
go mod download
```

2. Configure environment:
```bash
cp config/app.example.yaml config/app.yaml
# Edit config/app.yaml
```

3. Run migrations and start server:
```bash
make migrate
make run
```

## Documentation

API documentation available at `/api/docs` after starting the server.