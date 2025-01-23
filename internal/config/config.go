package config

import (
    "os"
    "strconv"
    "strings"
    "time"
    "fmt"

    "gopkg.in/yaml.v2"
)

type Config struct {
    App      AppConfig      `yaml:"app"`
    Database DatabaseConfig `yaml:"database"`
    Auth     AuthConfig     `yaml:"auth"`
    ML       MLConfig       `yaml:"ml"`
    Redis    RedisConfig    `yaml:"redis"`
    Services ServicesConfig `yaml:"services"`
}

type AppConfig struct {
    Name  string `yaml:"name"`
    Env   string `yaml:"env"`
    Port  int    `yaml:"port"`
    Debug bool   `yaml:"debug"`
}

type DatabaseConfig struct {
    Host     string `yaml:"host"`
    Port     int    `yaml:"port"`
    Name     string `yaml:"name"`
    User     string `yaml:"user"`
    Password string `yaml:"password"`
    SSLMode  string `yaml:"sslmode"`
}

type AuthConfig struct {
    JWTSecret     string        `yaml:"jwt_secret"`
    TokenExpiry   time.Duration `yaml:"token_expiry"`
    RefreshExpiry time.Duration `yaml:"refresh_expiry"`
}

type MLConfig struct {
    ModelPath      string        `yaml:"model_path"`
    UpdateInterval time.Duration `yaml:"update_interval"`
    BatchSize      int          `yaml:"batch_size"`
}

type RedisConfig struct {
    Host     string `yaml:"host"`
    Port     int    `yaml:"port"`
    Password string `yaml:"password"`
}

type ServicesConfig struct {
    MarketData MarketDataConfig `yaml:"market_data"`
}

type MarketDataConfig struct {
    Provider       string        `yaml:"provider"`
    APIKey        string        `yaml:"api_key"`
    UpdateInterval time.Duration `yaml:"update_interval"`
}

func Load(path string) (*Config, error) {
    file, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }

    var cfg Config
    if err := yaml.Unmarshal(file, &cfg); err != nil {
        return nil, err
    }

    if err := cfg.loadFromEnv(); err != nil {
        return nil, err
    }

    if err := cfg.validate(); err != nil {
        return nil, err
    }

    return &cfg, nil
}

func (c *Config) loadFromEnv() error {
    if env := os.Getenv("APP_ENV"); env != "" {
        c.App.Env = env
    }

    if port := os.Getenv("PORT"); port != "" {
        if p, err := strconv.Atoi(port); err == nil {
            c.App.Port = p
        }
    }

    if url := os.Getenv("DATABASE_URL"); url != "" {
        dbConfig, err := parseDatabaseURL(url)
        if err != nil {
            return err
        }
        c.Database = *dbConfig
    }

    if secret := os.Getenv("JWT_SECRET"); secret != "" {
        c.Auth.JWTSecret = secret
    }

    if apiKey := os.Getenv("MARKET_DATA_API_KEY"); apiKey != "" {
        c.Services.MarketData.APIKey = apiKey
    }

    return nil
}

func (c *Config) validate() error {
    if c.App.Port <= 0 || c.App.Port > 65535 {
        return fmt.Errorf("invalid port number: %d", c.App.Port)
    }

    if c.Database.Host == "" {
        return fmt.Errorf("database host is required")
    }

    if c.Auth.JWTSecret == "" {
        return fmt.Errorf("JWT secret is required")
    }

    if c.Services.MarketData.APIKey == "" {
        return fmt.Errorf("market data API key is required")
    }

    return nil
}

func parseDatabaseURL(url string) (*DatabaseConfig, error) {
    cfg := &DatabaseConfig{
        SSLMode: "disable",
    }

    // Remove postgresql:// prefix if present
    url = strings.TrimPrefix(url, "postgresql://")

    // Split credentials and host info
    parts := strings.Split(url, "@")
    if len(parts) != 2 {
        return nil, fmt.Errorf("invalid database URL format")
    }

    // Parse credentials
    credentials := strings.Split(parts[0], ":")
    if len(credentials) != 2 {
        return nil, fmt.Errorf("invalid credentials format")
    }
    cfg.User = credentials[0]
    cfg.Password = credentials[1]

    // Parse host, port, and database name
    hostInfo := strings.Split(parts[1], "/")
    if len(hostInfo) != 2 {
        return nil, fmt.Errorf("invalid host info format")
    }

    // Parse host and port
    hostPort := strings.Split(hostInfo[0], ":")
    if len(hostPort) != 2 {
        return nil, fmt.Errorf("invalid host/port format")
    }
    cfg.Host = hostPort[0]
    port, err := strconv.Atoi(hostPort[1])
    if err != nil {
        return nil, fmt.Errorf("invalid port number: %v", err)
    }
    cfg.Port = port

    // Parse database name and options
    dbNameOpts := strings.Split(hostInfo[1], "?")
    cfg.Name = dbNameOpts[0]

    // Parse options if present
    if len(dbNameOpts) > 1 {
        opts := strings.Split(dbNameOpts[1], "&")
        for _, opt := range opts {
            kv := strings.Split(opt, "=")
            if len(kv) == 2 && kv[0] == "sslmode" {
                cfg.SSLMode = kv[1]
            }
        }
    }

    return cfg, nil
}

func (c *Config) GetDatabaseURL() string {
    return fmt.Sprintf("postgresql://%s:%s@%s:%d/%s?sslmode=%s",
        c.Database.User,
        c.Database.Password,
        c.Database.Host,
        c.Database.Port,
        c.Database.Name,
        c.Database.SSLMode,
    )
}
