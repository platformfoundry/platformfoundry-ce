package cost

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"
)

// Forecaster predicts future infrastructure costs
type Forecaster struct {
	dataSource    DataSource
	models        map[string]ForecastModel
	defaultWindow time.Duration
}

// DataSource provides historical cost data
type DataSource interface {
	// GetCostData returns cost data points for a resource or category
	GetCostData(ctx context.Context, resource string, start, end time.Time) ([]CostDataPoint, error)
	// GetResourceList returns all resources with cost data
	GetResourceList(ctx context.Context) ([]string, error)
}

// ForecastModel represents a forecasting algorithm
type ForecastModel interface {
	Name() string
	Forecast(data []CostDataPoint, periods int) ([]ForecastPoint, error)
	Confidence() float64
}

// CostDataPoint represents a historical cost measurement
type CostDataPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Cost      float64   `json:"cost"`
	Resource  string    `json:"resource,omitempty"`
	Category  string    `json:"category,omitempty"`
	Tags      map[string]string `json:"tags,omitempty"`
}

// ForecastPoint represents a predicted future cost
type ForecastPoint struct {
	Timestamp  time.Time `json:"timestamp"`
	Predicted  float64   `json:"predicted"`
	LowerBound float64   `json:"lower_bound"`
	UpperBound float64   `json:"upper_bound"`
	Confidence float64   `json:"confidence"`
}

// CostForecast contains forecast results
type CostForecast struct {
	Resource        string                    `json:"resource"`
	Period          string                    `json:"period"`
	CurrentCost     float64                   `json:"current_cost"`
	PredictedCost   float64                   `json:"predicted_cost"`
	CostChange      float64                   `json:"cost_change"`
	CostChangePercent float64                 `json:"cost_change_percent"`
	Trend           string                    `json:"trend"` // increasing, stable, decreasing
	Confidence      float64                   `json:"confidence"`
	Forecasts       []ForecastPoint           `json:"forecasts"`
	BreakdownBy     map[string]float64        `json:"breakdown,omitempty"`
	Recommendations []CostRecommendation      `json:"recommendations,omitempty"`
	GeneratedAt     time.Time                 `json:"generated_at"`
}

// CostRecommendation suggests cost optimization actions
type CostRecommendation struct {
	Type            string  `json:"type"` // right_size, spot, reserved, delete
	Resource        string  `json:"resource"`
	CurrentCost     float64 `json:"current_cost"`
	PotentialSaving float64 `json:"potential_saving"`
	Confidence      float64 `json:"confidence"`
	Description     string  `json:"description"`
	Action          string  `json:"action"`
	Impact          string  `json:"impact"` // low, medium, high
}

// ForecasterConfig configures the cost forecaster
type ForecasterConfig struct {
	DataSource    DataSource
	DefaultWindow time.Duration
}

// NewForecaster creates a new cost forecaster
func NewForecaster(cfg ForecasterConfig) *Forecaster {
	window := cfg.DefaultWindow
	if window == 0 {
		window = 30 * 24 * time.Hour // Default 30 days
	}

	f := &Forecaster{
		dataSource:    cfg.DataSource,
		defaultWindow: window,
		models:        make(map[string]ForecastModel),
	}

	// Register default models
	f.models["linear"] = &LinearRegressionModel{}
	f.models["moving_average"] = &MovingAverageModel{Window: 7}
	f.models["exponential"] = &ExponentialSmoothingModel{Alpha: 0.3}

	return f
}

// Predict generates a cost forecast for a resource
func (f *Forecaster) Predict(ctx context.Context, resource string, periods int) (*CostForecast, error) {
	end := time.Now()
	start := end.Add(-f.defaultWindow)

	data, err := f.dataSource.GetCostData(ctx, resource, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to get cost data: %w", err)
	}

	if len(data) < 7 {
		return nil, fmt.Errorf("insufficient data points for forecasting (need at least 7)")
	}

	// Use the best performing model
	model := f.selectBestModel(data)

	forecasts, err := model.Forecast(data, periods)
	if err != nil {
		return nil, fmt.Errorf("forecast failed: %w", err)
	}

	// Calculate current and predicted costs
	currentCost := f.calculateAverageCost(data, 7) // Last 7 days
	predictedCost := f.calculateAverageForecast(forecasts)

	change := predictedCost - currentCost
	changePercent := 0.0
	if currentCost > 0 {
		changePercent = (change / currentCost) * 100
	}

	trend := determineTrend(changePercent)

	// Generate recommendations
	recommendations := f.generateRecommendations(ctx, resource, data, forecasts)

	return &CostForecast{
		Resource:          resource,
		Period:            fmt.Sprintf("%d days", periods),
		CurrentCost:       currentCost,
		PredictedCost:     predictedCost,
		CostChange:        change,
		CostChangePercent: changePercent,
		Trend:             trend,
		Confidence:        model.Confidence(),
		Forecasts:         forecasts,
		Recommendations:   recommendations,
		GeneratedAt:       time.Now(),
	}, nil
}

// PredictAll generates forecasts for all resources
func (f *Forecaster) PredictAll(ctx context.Context, periods int) ([]*CostForecast, error) {
	resources, err := f.dataSource.GetResourceList(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get resource list: %w", err)
	}

	forecasts := make([]*CostForecast, 0, len(resources))
	for _, resource := range resources {
		forecast, err := f.Predict(ctx, resource, periods)
		if err != nil {
			// Log but continue
			fmt.Printf("Warning: failed to forecast %s: %v\n", resource, err)
			continue
		}
		forecasts = append(forecasts, forecast)
	}

	return forecasts, nil
}

// GetTotalForecast returns an aggregate forecast
func (f *Forecaster) GetTotalForecast(ctx context.Context, periods int) (*CostForecast, error) {
	forecasts, err := f.PredictAll(ctx, periods)
	if err != nil {
		return nil, err
	}

	if len(forecasts) == 0 {
		return nil, fmt.Errorf("no forecasts available")
	}

	// Aggregate forecasts
	var totalCurrent, totalPredicted float64
	breakdown := make(map[string]float64)
	var allRecommendations []CostRecommendation

	for _, fc := range forecasts {
		totalCurrent += fc.CurrentCost
		totalPredicted += fc.PredictedCost
		breakdown[fc.Resource] = fc.PredictedCost
		allRecommendations = append(allRecommendations, fc.Recommendations...)
	}

	change := totalPredicted - totalCurrent
	changePercent := 0.0
	if totalCurrent > 0 {
		changePercent = (change / totalCurrent) * 100
	}

	// Sort recommendations by potential savings
	sort.Slice(allRecommendations, func(i, j int) bool {
		return allRecommendations[i].PotentialSaving > allRecommendations[j].PotentialSaving
	})

	// Keep top recommendations
	maxRecs := 10
	if len(allRecommendations) > maxRecs {
		allRecommendations = allRecommendations[:maxRecs]
	}

	return &CostForecast{
		Resource:          "total",
		Period:            fmt.Sprintf("%d days", periods),
		CurrentCost:       totalCurrent,
		PredictedCost:     totalPredicted,
		CostChange:        change,
		CostChangePercent: changePercent,
		Trend:             determineTrend(changePercent),
		Confidence:        0.8, // Aggregate confidence
		BreakdownBy:       breakdown,
		Recommendations:   allRecommendations,
		GeneratedAt:       time.Now(),
	}, nil
}

// selectBestModel chooses the best forecasting model for the data
func (f *Forecaster) selectBestModel(data []CostDataPoint) ForecastModel {
	// For now, use exponential smoothing as default
	// In production, this would cross-validate models
	return f.models["exponential"]
}

// calculateAverageCost calculates average cost over the last N days
func (f *Forecaster) calculateAverageCost(data []CostDataPoint, days int) float64 {
	if len(data) == 0 {
		return 0
	}

	cutoff := time.Now().Add(-time.Duration(days) * 24 * time.Hour)
	var sum float64
	var count int

	for _, d := range data {
		if d.Timestamp.After(cutoff) {
			sum += d.Cost
			count++
		}
	}

	if count == 0 {
		return 0
	}
	return sum / float64(count)
}

// calculateAverageForecast calculates average forecasted cost
func (f *Forecaster) calculateAverageForecast(forecasts []ForecastPoint) float64 {
	if len(forecasts) == 0 {
		return 0
	}

	var sum float64
	for _, fc := range forecasts {
		sum += fc.Predicted
	}
	return sum / float64(len(forecasts))
}

// generateRecommendations creates cost optimization suggestions
func (f *Forecaster) generateRecommendations(ctx context.Context, resource string, data []CostDataPoint, forecasts []ForecastPoint) []CostRecommendation {
	recommendations := make([]CostRecommendation, 0)

	if len(data) == 0 {
		return recommendations
	}

	avgCost := f.calculateAverageCost(data, 30)

	// Check for underutilization (simulated check)
	// In production, this would check actual utilization metrics
	if avgCost > 100 {
		recommendations = append(recommendations, CostRecommendation{
			Type:            "right_size",
			Resource:        resource,
			CurrentCost:     avgCost,
			PotentialSaving: avgCost * 0.3,
			Confidence:      0.75,
			Description:     fmt.Sprintf("Resource %s may be over-provisioned", resource),
			Action:          "Review resource sizing and consider downsizing",
			Impact:          "medium",
		})
	}

	// Check for spot instance opportunity
	if avgCost > 200 {
		recommendations = append(recommendations, CostRecommendation{
			Type:            "spot",
			Resource:        resource,
			CurrentCost:     avgCost,
			PotentialSaving: avgCost * 0.6,
			Confidence:      0.8,
			Description:     "Consider using spot instances for fault-tolerant workloads",
			Action:          "Migrate batch workloads to spot instances",
			Impact:          "high",
		})
	}

	// Check for reserved instance opportunity
	if avgCost > 500 {
		recommendations = append(recommendations, CostRecommendation{
			Type:            "reserved",
			Resource:        resource,
			CurrentCost:     avgCost,
			PotentialSaving: avgCost * 0.4,
			Confidence:      0.9,
			Description:     "Consistent usage pattern suggests reserved instance savings",
			Action:          "Purchase reserved capacity for 1 or 3 year term",
			Impact:          "high",
		})
	}

	return recommendations
}

// determineTrend classifies the cost trend
func determineTrend(changePercent float64) string {
	if changePercent > 5 {
		return "increasing"
	} else if changePercent < -5 {
		return "decreasing"
	}
	return "stable"
}

// LinearRegressionModel implements simple linear regression
type LinearRegressionModel struct{}

func (m *LinearRegressionModel) Name() string { return "linear" }

func (m *LinearRegressionModel) Confidence() float64 { return 0.7 }

func (m *LinearRegressionModel) Forecast(data []CostDataPoint, periods int) ([]ForecastPoint, error) {
	n := float64(len(data))
	if n < 2 {
		return nil, fmt.Errorf("insufficient data for linear regression")
	}

	// Calculate linear regression coefficients
	var sumX, sumY, sumXY, sumX2 float64
	for i, d := range data {
		x := float64(i)
		y := d.Cost
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}

	slope := (n*sumXY - sumX*sumY) / (n*sumX2 - sumX*sumX)
	intercept := (sumY - slope*sumX) / n

	// Generate forecasts
	forecasts := make([]ForecastPoint, periods)
	baseTime := data[len(data)-1].Timestamp
	interval := 24 * time.Hour

	for i := 0; i < periods; i++ {
		x := float64(len(data) + i)
		predicted := slope*x + intercept
		if predicted < 0 {
			predicted = 0
		}

		// Calculate confidence interval (simplified)
		margin := predicted * 0.1

		forecasts[i] = ForecastPoint{
			Timestamp:  baseTime.Add(time.Duration(i+1) * interval),
			Predicted:  predicted,
			LowerBound: predicted - margin,
			UpperBound: predicted + margin,
			Confidence: 0.7,
		}
	}

	return forecasts, nil
}

// MovingAverageModel implements simple moving average
type MovingAverageModel struct {
	Window int
}

func (m *MovingAverageModel) Name() string { return "moving_average" }

func (m *MovingAverageModel) Confidence() float64 { return 0.65 }

func (m *MovingAverageModel) Forecast(data []CostDataPoint, periods int) ([]ForecastPoint, error) {
	if len(data) < m.Window {
		return nil, fmt.Errorf("insufficient data for moving average")
	}

	// Calculate moving average of last N points
	var sum float64
	for i := len(data) - m.Window; i < len(data); i++ {
		sum += data[i].Cost
	}
	avg := sum / float64(m.Window)

	// Generate forecasts (constant prediction)
	forecasts := make([]ForecastPoint, periods)
	baseTime := data[len(data)-1].Timestamp
	interval := 24 * time.Hour

	for i := 0; i < periods; i++ {
		margin := avg * 0.15
		forecasts[i] = ForecastPoint{
			Timestamp:  baseTime.Add(time.Duration(i+1) * interval),
			Predicted:  avg,
			LowerBound: avg - margin,
			UpperBound: avg + margin,
			Confidence: 0.65,
		}
	}

	return forecasts, nil
}

// ExponentialSmoothingModel implements simple exponential smoothing
type ExponentialSmoothingModel struct {
	Alpha float64 // Smoothing factor (0 < alpha <= 1)
}

func (m *ExponentialSmoothingModel) Name() string { return "exponential" }

func (m *ExponentialSmoothingModel) Confidence() float64 { return 0.8 }

func (m *ExponentialSmoothingModel) Forecast(data []CostDataPoint, periods int) ([]ForecastPoint, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("no data for exponential smoothing")
	}

	// Initialize with first observation
	smoothed := data[0].Cost

	// Apply exponential smoothing
	for i := 1; i < len(data); i++ {
		smoothed = m.Alpha*data[i].Cost + (1-m.Alpha)*smoothed
	}

	// Calculate trend
	var trend float64
	if len(data) > 1 {
		recentAvg := (data[len(data)-1].Cost + data[len(data)-2].Cost) / 2
		olderAvg := (data[0].Cost + data[1].Cost) / 2
		trend = (recentAvg - olderAvg) / float64(len(data))
	}

	// Generate forecasts
	forecasts := make([]ForecastPoint, periods)
	baseTime := data[len(data)-1].Timestamp
	interval := 24 * time.Hour

	for i := 0; i < periods; i++ {
		predicted := smoothed + trend*float64(i+1)
		if predicted < 0 {
			predicted = 0
		}

		// Calculate confidence interval
		margin := predicted * (0.1 + 0.01*float64(i)) // Wider margin for further predictions

		forecasts[i] = ForecastPoint{
			Timestamp:  baseTime.Add(time.Duration(i+1) * interval),
			Predicted:  math.Round(predicted*100) / 100,
			LowerBound: math.Round((predicted-margin)*100) / 100,
			UpperBound: math.Round((predicted+margin)*100) / 100,
			Confidence: math.Max(0.5, 0.8-0.02*float64(i)),
		}
	}

	return forecasts, nil
}

// MockDataSource provides simulated cost data for testing
type MockDataSource struct {
	data map[string][]CostDataPoint
}

// NewMockDataSource creates a mock data source with sample data
func NewMockDataSource() *MockDataSource {
	ds := &MockDataSource{
		data: make(map[string][]CostDataPoint),
	}

	// Generate sample data
	resources := []string{
		"compute/api-gateway",
		"compute/order-service",
		"database/orders-db",
		"storage/logs-bucket",
		"network/load-balancer",
	}

	baseCosts := map[string]float64{
		"compute/api-gateway":   450.0,
		"compute/order-service": 320.0,
		"database/orders-db":    580.0,
		"storage/logs-bucket":   120.0,
		"network/load-balancer": 85.0,
	}

	now := time.Now()
	for _, r := range resources {
		base := baseCosts[r]
		points := make([]CostDataPoint, 90) // 90 days of data

		for i := 0; i < 90; i++ {
			// Add some variance and trend
			variance := (float64(i%7) - 3) * (base * 0.02)
			trend := float64(i) * (base * 0.001)
			cost := base + variance + trend

			points[i] = CostDataPoint{
				Timestamp: now.Add(-time.Duration(90-i) * 24 * time.Hour),
				Cost:      cost,
				Resource:  r,
			}
		}
		ds.data[r] = points
	}

	return ds
}

func (ds *MockDataSource) GetCostData(ctx context.Context, resource string, start, end time.Time) ([]CostDataPoint, error) {
	data, ok := ds.data[resource]
	if !ok {
		return nil, fmt.Errorf("resource not found: %s", resource)
	}

	// Filter by time range
	result := make([]CostDataPoint, 0)
	for _, d := range data {
		if (d.Timestamp.After(start) || d.Timestamp.Equal(start)) &&
			(d.Timestamp.Before(end) || d.Timestamp.Equal(end)) {
			result = append(result, d)
		}
	}

	return result, nil
}

func (ds *MockDataSource) GetResourceList(ctx context.Context) ([]string, error) {
	resources := make([]string, 0, len(ds.data))
	for r := range ds.data {
		resources = append(resources, r)
	}
	return resources, nil
}
