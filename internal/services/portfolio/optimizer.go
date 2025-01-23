package portfolio

import (
    "context"
    "database/sql"
    "gonum.org/v1/gonum/mat"
    "gonum.org/v1/gonum/optimize"
    "math"
)

type PortfolioOptimizer struct {
    db *sql.DB
    riskFreeRate float64
    minWeight    float64
    maxWeight    float64
}

type OptimizationResult struct {
    Weights        []float64 `json:"weights"`
    ExpectedReturn float64   `json:"expected_return"`
    Risk          float64   `json:"risk"`
    SharpeRatio   float64   `json:"sharpe_ratio"`
}

func NewPortfolioOptimizer(db *sql.DB) *PortfolioOptimizer {
    return &PortfolioOptimizer{
        db:           db,
        riskFreeRate: 0.02, // 2% risk-free rate
        minWeight:    0.0,  // minimum weight per asset
        maxWeight:    0.4,  // maximum weight per asset (40%)
    }
}

func (o *PortfolioOptimizer) Optimize(ctx context.Context, symbols []string, riskTolerance float64) (*OptimizationResult, error) {
    // Get historical returns
    returns, err := o.getHistoricalReturns(ctx, symbols)
    if err != nil {
        return nil, err
    }

    // Calculate expected returns and covariance matrix
    expectedReturns := o.calculateExpectedReturns(returns)
    covMatrix := o.calculateCovarianceMatrix(returns)

    // Initialize optimization problem
    n := len(symbols)
    weights := make([]float64, n)
    for i := range weights {
        weights[i] = 1.0 / float64(n) // Start with equal weights
    }

    // Define optimization problem
    problem := optimize.Problem{
        Func: func(w []float64) float64 {
            return o.objectiveFunction(w, expectedReturns, covMatrix, riskTolerance)
        },
        Grad: func(grad, w []float64) {
            o.calculateGradient(grad, w, expectedReturns, covMatrix, riskTolerance)
        },
    }

    // Set constraints
    fc := optimize.FunctionConstraint{
        Func: func(w []float64) float64 {
            sum := 0.0
            for _, weight := range w {
                sum += weight
            }
            return sum - 1.0 // Sum of weights should be 1
        },
    }

    // Run optimization
    result, err := optimize.Minimize(problem, weights, nil, nil)
    if err != nil {
        return nil, err
    }

    // Calculate metrics for optimized portfolio
    optimizedWeights := result.X
    portfolioReturn := o.calculatePortfolioReturn(optimizedWeights, expectedReturns)
    portfolioRisk := o.calculatePortfolioRisk(optimizedWeights, covMatrix)
    sharpeRatio := (portfolioReturn - o.riskFreeRate) / portfolioRisk

    return &OptimizationResult{
        Weights:        optimizedWeights,
        ExpectedReturn: portfolioReturn,
        Risk:          portfolioRisk,
        SharpeRatio:   sharpeRatio,
    }, nil
}

func (o *PortfolioOptimizer) getHistoricalReturns(ctx context.Context, symbols []string) ([][]float64, error) {
    query := `
        WITH daily_returns AS (
            SELECT 
                symbol,
                timestamp,
                (close - LAG(close) OVER (PARTITION BY symbol ORDER BY timestamp)) / LAG(close) OVER (PARTITION BY symbol ORDER BY timestamp) as return
            FROM market_data
            WHERE symbol = ANY($1)
            AND timestamp >= NOW() - INTERVAL '1 year'
            ORDER BY timestamp
        )
        SELECT symbol, ARRAY_AGG(return ORDER BY timestamp) as returns
        FROM daily_returns
        WHERE return IS NOT NULL
        GROUP BY symbol
    `

    rows, err := o.db.QueryContext(ctx, query, symbols)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    returns := make([][]float64, len(symbols))
    for rows.Next() {
        var symbol string
        var symbolReturns []float64
        if err := rows.Scan(&symbol, &symbolReturns); err != nil {
            return nil, err
        }

        for i, s := range symbols {
            if s == symbol {
                returns[i] = symbolReturns
                break
            }
        }
    }

    return returns, nil
}

func (o *PortfolioOptimizer) calculateExpectedReturns(returns [][]float64) []float64 {
    n := len(returns)
    expectedReturns := make([]float64, n)

    for i := 0; i < n; i++ {
        sum := 0.0
        for _, r := range returns[i] {
            sum += r
        }
        expectedReturns[i] = sum / float64(len(returns[i]))
    }

    return expectedReturns
}

func (o *PortfolioOptimizer) calculateCovarianceMatrix(returns [][]float64) *mat.Dense {
    n := len(returns)
    numPeriods := len(returns[0])
    
    // Create matrices for calculation
    data := mat.NewDense(numPeriods, n, nil)
    for i := 0; i < n; i++ {
        for j := 0; j < numPeriods; j++ {
            data.Set(j, i, returns[i][j])
        }
    }

    // Calculate covariance matrix
    var covMatrix mat.Dense
    covMatrix.Scale(1/float64(numPeriods-1), data.T().Mul(data))
    
    return &covMatrix
}

func (o *PortfolioOptimizer) objectiveFunction(weights []float64, expectedReturns []float64, covMatrix *mat.Dense, riskTolerance float64) float64 {
    portfolioReturn := o.calculatePortfolioReturn(weights, expectedReturns)
    portfolioRisk := o.calculatePortfolioRisk(weights, covMatrix)
    
    // Objective: Maximize Sharpe Ratio
    // For minimization, we return negative Sharpe ratio
    return -(portfolioReturn - o.riskFreeRate) / portfolioRisk
}

func (o *PortfolioOptimizer) calculateGradient(grad, weights []float64, expectedReturns []float64, covMatrix *mat.Dense, riskTolerance float64) {
    n := len(weights)
    h := 1e-8 // Small value for numerical gradient calculation

    // Calculate gradient numerically
    for i := 0; i < n; i++ {
        weightsPlus := make([]float64, n)
        weightsMinus := make([]float64, n)
        copy(weightsPlus, weights)
        copy(weightsMinus, weights)
        
        weightsPlus[i] += h
        weightsMinus[i] -= h

        fPlus := o.objectiveFunction(weightsPlus, expectedReturns, covMatrix, riskTolerance)
        fMinus := o.objectiveFunction(weightsMinus, expectedReturns, covMatrix, riskTolerance)
        
        grad[i] = (fPlus - fMinus) / (2 * h)
    }
}

func (o *PortfolioOptimizer) calculatePortfolioReturn(weights []float64, expectedReturns []float64) float64 {
    var sum float64
    for i, w := range weights {
        sum += w * expectedReturns[i]
    }
    return sum
}

func (o *PortfolioOptimizer) calculatePortfolioRisk(weights []float64, covMatrix *mat.Dense) float64 {
    n := len(weights)
    var variance float64

    for i := 0; i < n; i++ {
        for j := 0; j < n; j++ {
            variance += weights[i] * weights[j] * covMatrix.At(i, j)
        }
    }

    return math.Sqrt(variance)
}
