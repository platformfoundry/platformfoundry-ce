package scaling

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"
)

// MetricsProvider interface for fetching metrics
type MetricsProvider interface {
	// Query fetches metric values for a time range
	Query(ctx context.Context, query string, start, end time.Time, step time.Duration) ([]DataPoint, error)
	// GetCurrent fetches current metric value
	GetCurrent(ctx context.Context, query string) (float64, error)
}

// DataPoint represents a single metric data point
type DataPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}

// Predictor performs time-series predictions for scaling
type Predictor struct {
	config           *PredictorConfig
	metricsProvider  MetricsProvider
	models           map[string]*TrainedModel
	patterns         map[string]*TrafficPattern
	mu               sync.RWMutex
}

// PredictorConfig configures the predictor
type PredictorConfig struct {
	DefaultModel       PredictionModel `json:"defaultModel"`
	TrainingWindow     time.Duration   `json:"trainingWindow"`
	MinDataPoints      int             `json:"minDataPoints"`
	UpdateInterval     time.Duration   `json:"updateInterval"`
	ConfidenceThreshold float64        `json:"confidenceThreshold"`
}

// DefaultPredictorConfig returns default predictor configuration
func DefaultPredictorConfig() *PredictorConfig {
	return &PredictorConfig{
		DefaultModel:       ModelHoltWinter,
		TrainingWindow:     7 * 24 * time.Hour, // 7 days
		MinDataPoints:      100,
		UpdateInterval:     1 * time.Hour,
		ConfidenceThreshold: 0.7,
	}
}

// TrainedModel holds a trained prediction model
type TrainedModel struct {
	Name          string          `json:"name"`
	ModelType     PredictionModel `json:"modelType"`
	TrainedAt     time.Time       `json:"trainedAt"`
	DataPoints    int             `json:"dataPoints"`
	Coefficients  []float64       `json:"coefficients,omitempty"`
	Seasonality   *SeasonalPattern `json:"seasonality,omitempty"`
	Trend         float64         `json:"trend,omitempty"`
	Level         float64         `json:"level,omitempty"`
	MAE           float64         `json:"mae"`
	RMSE          float64         `json:"rmse"`
}

// NewPredictor creates a new predictor
func NewPredictor(config *PredictorConfig, provider MetricsProvider) *Predictor {
	if config == nil {
		config = DefaultPredictorConfig()
	}
	return &Predictor{
		config:          config,
		metricsProvider: provider,
		models:          make(map[string]*TrainedModel),
		patterns:        make(map[string]*TrafficPattern),
	}
}

// Train trains a prediction model for a metric
func (p *Predictor) Train(ctx context.Context, name string, query string, modelType PredictionModel) (*TrainedModel, error) {
	if p.metricsProvider == nil {
		return nil, fmt.Errorf("no metrics provider configured")
	}

	// Fetch historical data
	end := time.Now()
	start := end.Add(-p.config.TrainingWindow)
	step := 5 * time.Minute

	data, err := p.metricsProvider.Query(ctx, query, start, end, step)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch training data: %w", err)
	}

	if len(data) < p.config.MinDataPoints {
		return nil, fmt.Errorf("insufficient data points: got %d, need %d", len(data), p.config.MinDataPoints)
	}

	// Train model based on type
	var model *TrainedModel
	switch modelType {
	case ModelHoltWinter:
		model, err = p.trainHoltWinters(name, data)
	case ModelLinear:
		model, err = p.trainLinear(name, data)
	case ModelARIMA:
		model, err = p.trainARIMA(name, data)
	default:
		model, err = p.trainHoltWinters(name, data)
	}

	if err != nil {
		return nil, fmt.Errorf("training failed: %w", err)
	}

	// Store model
	p.mu.Lock()
	p.models[name] = model
	p.mu.Unlock()

	// Learn traffic patterns
	pattern := p.learnPattern(name, data)
	p.mu.Lock()
	p.patterns[name] = pattern
	p.mu.Unlock()

	return model, nil
}

// Predict generates predictions for the given horizon
func (p *Predictor) Predict(ctx context.Context, name string, horizon time.Duration) (*PredictionResult, error) {
	p.mu.RLock()
	model, ok := p.models[name]
	p.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("no trained model found for %s", name)
	}

	// Generate forecast points
	now := time.Now()
	step := 5 * time.Minute
	numPoints := int(horizon / step)
	if numPoints < 1 {
		numPoints = 1
	}

	forecast := make([]ForecastPoint, numPoints)
	var maxPredicted float64

	for i := 0; i < numPoints; i++ {
		timestamp := now.Add(time.Duration(i) * step)
		value, lowerBound, upperBound := p.predictPoint(model, timestamp, i)

		forecast[i] = ForecastPoint{
			Timestamp:  timestamp,
			Value:      value,
			LowerBound: lowerBound,
			UpperBound: upperBound,
		}

		if value > maxPredicted {
			maxPredicted = value
		}
	}

	// Calculate confidence based on model metrics
	confidence := p.calculateConfidence(model)

	return &PredictionResult{
		Timestamp:       now,
		Horizon:         horizon.String(),
		PredictedLoad:   maxPredicted,
		ConfidenceLevel: confidence,
		Forecast:        forecast,
		ModelMetrics: &ModelMetrics{
			MAE:         model.MAE,
			RMSE:        model.RMSE,
			MAPE:        model.MAE / (model.Level + 1) * 100,
			TrainingAge: time.Since(model.TrainedAt).String(),
		},
	}, nil
}

// GetPattern returns learned traffic pattern
func (p *Predictor) GetPattern(name string) (*TrafficPattern, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	pattern, ok := p.patterns[name]
	return pattern, ok
}

// trainHoltWinters implements Holt-Winters exponential smoothing
func (p *Predictor) trainHoltWinters(name string, data []DataPoint) (*TrainedModel, error) {
	n := len(data)
	if n < 48 { // Need at least 2 days of hourly data
		return p.trainLinear(name, data) // Fallback to linear
	}

	// Extract values
	values := make([]float64, n)
	for i, dp := range data {
		values[i] = dp.Value
	}

	// Parameters
	alpha := 0.3 // Level smoothing
	beta := 0.1  // Trend smoothing
	gamma := 0.1 // Seasonal smoothing
	seasonLength := 288 // 24 hours at 5-minute intervals

	if n < seasonLength*2 {
		seasonLength = 24 // Hourly seasonality
	}

	// Initialize components
	level := mean(values[:seasonLength])
	trend := (mean(values[seasonLength:2*seasonLength]) - mean(values[:seasonLength])) / float64(seasonLength)

	seasonal := make([]float64, seasonLength)
	for i := 0; i < seasonLength; i++ {
		seasonal[i] = values[i] / level
	}

	// Fit model
	var sumError float64
	for i := seasonLength; i < n; i++ {
		seasonIndex := i % seasonLength
		lastLevel := level

		// Update level
		level = alpha*(values[i]/seasonal[seasonIndex]) + (1-alpha)*(level+trend)

		// Update trend
		trend = beta*(level-lastLevel) + (1-beta)*trend

		// Update seasonal
		seasonal[seasonIndex] = gamma*(values[i]/level) + (1-gamma)*seasonal[seasonIndex]

		// Calculate error
		predicted := (lastLevel + trend) * seasonal[seasonIndex]
		sumError += math.Abs(values[i] - predicted)
	}

	mae := sumError / float64(n-seasonLength)

	return &TrainedModel{
		Name:         name,
		ModelType:    ModelHoltWinter,
		TrainedAt:    time.Now(),
		DataPoints:   n,
		Level:        level,
		Trend:        trend,
		Coefficients: []float64{alpha, beta, gamma},
		Seasonality: &SeasonalPattern{
			Daily: seasonal,
		},
		MAE:  mae,
		RMSE: mae * 1.25, // Approximation
	}, nil
}

// trainLinear implements simple linear regression
func (p *Predictor) trainLinear(name string, data []DataPoint) (*TrainedModel, error) {
	n := len(data)
	if n < 2 {
		return nil, fmt.Errorf("insufficient data for linear regression")
	}

	// Calculate means
	var sumX, sumY, sumXY, sumX2 float64
	for i, dp := range data {
		x := float64(i)
		y := dp.Value
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}

	nf := float64(n)
	slope := (nf*sumXY - sumX*sumY) / (nf*sumX2 - sumX*sumX)
	intercept := (sumY - slope*sumX) / nf

	// Calculate MAE
	var sumError float64
	for i, dp := range data {
		predicted := intercept + slope*float64(i)
		sumError += math.Abs(dp.Value - predicted)
	}
	mae := sumError / nf

	return &TrainedModel{
		Name:         name,
		ModelType:    ModelLinear,
		TrainedAt:    time.Now(),
		DataPoints:   n,
		Level:        intercept,
		Trend:        slope,
		Coefficients: []float64{intercept, slope},
		MAE:          mae,
		RMSE:         mae * 1.25,
	}, nil
}

// trainARIMA implements simplified ARIMA model
func (p *Predictor) trainARIMA(name string, data []DataPoint) (*TrainedModel, error) {
	// Simplified AR(1) model
	n := len(data)
	if n < 10 {
		return p.trainLinear(name, data)
	}

	values := make([]float64, n)
	for i, dp := range data {
		values[i] = dp.Value
	}

	// Calculate AR(1) coefficient
	meanVal := mean(values)
	var numerator, denominator float64
	for i := 1; i < n; i++ {
		numerator += (values[i] - meanVal) * (values[i-1] - meanVal)
		denominator += (values[i-1] - meanVal) * (values[i-1] - meanVal)
	}
	phi := numerator / denominator
	if math.IsNaN(phi) || math.IsInf(phi, 0) {
		phi = 0.5
	}

	// Calculate error
	var sumError float64
	for i := 1; i < n; i++ {
		predicted := meanVal + phi*(values[i-1]-meanVal)
		sumError += math.Abs(values[i] - predicted)
	}
	mae := sumError / float64(n-1)

	return &TrainedModel{
		Name:         name,
		ModelType:    ModelARIMA,
		TrainedAt:    time.Now(),
		DataPoints:   n,
		Level:        meanVal,
		Coefficients: []float64{phi},
		MAE:          mae,
		RMSE:         mae * 1.25,
	}, nil
}

// predictPoint predicts a single point
func (p *Predictor) predictPoint(model *TrainedModel, timestamp time.Time, steps int) (value, lower, upper float64) {
	switch model.ModelType {
	case ModelHoltWinter:
		// Holt-Winters prediction
		value = model.Level + model.Trend*float64(steps+1)
		if model.Seasonality != nil && len(model.Seasonality.Daily) > 0 {
			hour := timestamp.Hour()
			minute := timestamp.Minute()
			seasonIndex := (hour*60 + minute) / 5 // 5-minute intervals
			if seasonIndex < len(model.Seasonality.Daily) {
				value *= model.Seasonality.Daily[seasonIndex]
			}
		}
	case ModelLinear:
		value = model.Level + model.Trend*float64(steps+1)
	case ModelARIMA:
		phi := model.Coefficients[0]
		value = model.Level + math.Pow(phi, float64(steps+1))*(model.Level-model.Level)
		if value < 0 {
			value = model.Level
		}
	default:
		value = model.Level
	}

	// Calculate confidence interval
	uncertainty := model.MAE * math.Sqrt(float64(steps+1)) * 1.96
	lower = value - uncertainty
	upper = value + uncertainty

	if lower < 0 {
		lower = 0
	}

	return value, lower, upper
}

// learnPattern learns traffic patterns from historical data
func (p *Predictor) learnPattern(name string, data []DataPoint) *TrafficPattern {
	pattern := &TrafficPattern{
		Name:        name,
		DayOfWeek:   make([]DayPattern, 7),
		LastUpdated: time.Now(),
	}

	// Group data by day of week and hour
	dayHourValues := make(map[int]map[int][]float64)
	for i := 0; i < 7; i++ {
		dayHourValues[i] = make(map[int][]float64)
		for h := 0; h < 24; h++ {
			dayHourValues[i][h] = make([]float64, 0)
		}
	}

	for _, dp := range data {
		day := int(dp.Timestamp.Weekday())
		hour := dp.Timestamp.Hour()
		dayHourValues[day][hour] = append(dayHourValues[day][hour], dp.Value)
	}

	// Calculate patterns
	days := []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}
	for d := 0; d < 7; d++ {
		hourlyLoad := make([]float64, 24)
		for h := 0; h < 24; h++ {
			if len(dayHourValues[d][h]) > 0 {
				hourlyLoad[h] = mean(dayHourValues[d][h])
			}
		}

		// Find peak and trough hours
		peakHours := findPeakHours(hourlyLoad, 3)
		troughHours := findTroughHours(hourlyLoad, 3)

		pattern.DayOfWeek[d] = DayPattern{
			Day:         days[d],
			HourlyLoad:  hourlyLoad,
			PeakHours:   peakHours,
			TroughHours: troughHours,
		}
	}

	return pattern
}

// calculateConfidence calculates prediction confidence
func (p *Predictor) calculateConfidence(model *TrainedModel) float64 {
	// Base confidence on model age and accuracy
	ageHours := time.Since(model.TrainedAt).Hours()
	ageFactor := math.Max(0, 1-ageHours/168) // Decreases over a week

	// Accuracy factor based on RMSE relative to level
	accuracyFactor := 1.0
	if model.Level > 0 {
		relativeError := model.RMSE / model.Level
		accuracyFactor = math.Max(0, 1-relativeError)
	}

	// Data points factor
	dataFactor := math.Min(1, float64(model.DataPoints)/1000)

	confidence := (ageFactor*0.3 + accuracyFactor*0.5 + dataFactor*0.2)
	return math.Min(1, math.Max(0, confidence))
}

// Helper functions

func mean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	var sum float64
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func findPeakHours(hourlyLoad []float64, n int) []int {
	type hourValue struct {
		hour  int
		value float64
	}
	hours := make([]hourValue, len(hourlyLoad))
	for i, v := range hourlyLoad {
		hours[i] = hourValue{hour: i, value: v}
	}
	sort.Slice(hours, func(i, j int) bool {
		return hours[i].value > hours[j].value
	})

	result := make([]int, 0, n)
	for i := 0; i < n && i < len(hours); i++ {
		result = append(result, hours[i].hour)
	}
	sort.Ints(result)
	return result
}

func findTroughHours(hourlyLoad []float64, n int) []int {
	type hourValue struct {
		hour  int
		value float64
	}
	hours := make([]hourValue, len(hourlyLoad))
	for i, v := range hourlyLoad {
		hours[i] = hourValue{hour: i, value: v}
	}
	sort.Slice(hours, func(i, j int) bool {
		return hours[i].value < hours[j].value
	})

	result := make([]int, 0, n)
	for i := 0; i < n && i < len(hours); i++ {
		if hours[i].value > 0 { // Only include non-zero values
			result = append(result, hours[i].hour)
		}
	}
	sort.Ints(result)
	return result
}
