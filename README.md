# QuantAI - AI-Powered Investment Platform

QuantAI is a sophisticated investment platform that leverages artificial intelligence to provide advanced portfolio management and market analysis capabilities.

## Features

- 🤖 AI-Powered Market Analysis
- 📊 Portfolio Management & Optimization
- 📈 Real-time Trading Signals
- 🔮 Predictive Analytics
- 🎯 Custom Investment Strategies
- 📱 Multi-Platform Support

## Technology Stack

- **Backend**: Go
- **Database**: PostgreSQL
- **Authentication**: JWT
- **AI/ML**: Custom machine learning models
- **API**: RESTful with JSON

## Getting Started

### Prerequisites

- Go 1.21 or higher
- PostgreSQL 14 or higher
- Make (optional, for using Makefile commands)

### Installation

1. Clone the repository:
```bash
git clone https://github.com/Cryptoprojectsfun/quantai-clone.git
cd quantai-clone
```

2. Install dependencies:
```bash
go mod download
```

3. Set up the environment variables:
```bash
cp .env.example .env
# Edit .env with your configuration
```

4. Run database migrations:
```bash
make migrate-up
```

5. Start the server:
```bash
make run
```

### Configuration

The application can be configured using environment variables or a .env file:

```env
ENVIRONMENT=development
PORT=8080
DATABASE_URL=postgresql://user:password@localhost:5432/quantai?sslmode=disable
JWT_SECRET=your-secret-key
LOG_LEVEL=debug
RATE_LIMIT=100
```

## Project Structure

```
.
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── models/
│   │   ├── user.go
│   │   ├── portfolio.go
│   │   └── prediction.go
│   ├── services/
│   │   ├── ai/
│   │   ├── portfolio/
│   │   └── analytics/
│   └── middleware/
│       └── auth.go
├── pkg/
│   ├── logger/
│   └── database/
└── migrations/
```
