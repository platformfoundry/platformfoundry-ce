package cost

import (
	"encoding/json"
	"fmt"
	"os"
)

// Pricing represents pricing information for a provider/region/resource type
type Pricing struct {
	Provider            Provider     `json:"provider"`
	Region              string       `json:"region"`
	ResourceType        ResourceType `json:"resourceType"`
	Currency            string       `json:"currency"`
	CPUPerHour          float64      `json:"cpuPerHour"`
	MemoryGBPerHour     float64      `json:"memoryGBPerHour"`
	StorageGBPerMonth   float64      `json:"storageGBPerMonth"`
	LoadBalancerPerHour float64      `json:"loadBalancerPerHour"`
	K8sClusterPerHour   float64      `json:"k8sClusterPerHour"`
	NetworkGBPerMonth   float64      `json:"networkGBPerMonth"`
}

// PricingDB represents a pricing database
type PricingDB struct {
	Prices map[string]*Pricing `json:"prices"`
}

// NewDefaultPricingDB creates a pricing database with default prices
func NewDefaultPricingDB() *PricingDB {
	db := &PricingDB{
		Prices: make(map[string]*Pricing),
	}

	// AWS Pricing (us-east-1)
	db.Prices["aws:us-east-1:compute"] = &Pricing{
		Provider:          ProviderAWS,
		Region:            "us-east-1",
		ResourceType:      ResourceTypeCompute,
		Currency:          "USD",
		CPUPerHour:        0.05, // ~$36/month per vCPU
		MemoryGBPerHour:   0.01, // ~$7.3/month per GB
		StorageGBPerMonth: 0.10, // EBS pricing
		K8sClusterPerHour: 0.10, // EKS control plane
	}

	db.Prices["aws:us-west-2:compute"] = &Pricing{
		Provider:          ProviderAWS,
		Region:            "us-west-2",
		ResourceType:      ResourceTypeCompute,
		Currency:          "USD",
		CPUPerHour:        0.052,
		MemoryGBPerHour:   0.0105,
		StorageGBPerMonth: 0.10,
		K8sClusterPerHour: 0.10,
	}

	db.Prices["aws:us-east-1:storage"] = &Pricing{
		Provider:          ProviderAWS,
		Region:            "us-east-1",
		ResourceType:      ResourceTypeStorage,
		Currency:          "USD",
		StorageGBPerMonth: 0.023, // S3 standard
	}

	db.Prices["aws:us-east-1:database"] = &Pricing{
		Provider:          ProviderAWS,
		Region:            "us-east-1",
		ResourceType:      ResourceTypeDatabase,
		Currency:          "USD",
		CPUPerHour:        0.075,
		MemoryGBPerHour:   0.015,
		StorageGBPerMonth: 0.115, // RDS pricing
	}

	db.Prices["aws:us-east-1:load_balancer"] = &Pricing{
		Provider:            ProviderAWS,
		Region:              "us-east-1",
		ResourceType:        ResourceTypeLoadBalancer,
		Currency:            "USD",
		LoadBalancerPerHour: 0.025,
	}

	// GCP Pricing (us-central1)
	db.Prices["gcp:us-central1:compute"] = &Pricing{
		Provider:          ProviderGCP,
		Region:            "us-central1",
		ResourceType:      ResourceTypeCompute,
		Currency:          "USD",
		CPUPerHour:        0.048,
		MemoryGBPerHour:   0.0095,
		StorageGBPerMonth: 0.08, // Persistent disk
		K8sClusterPerHour: 0.10, // GKE control plane
	}

	db.Prices["gcp:us-central1:storage"] = &Pricing{
		Provider:          ProviderGCP,
		Region:            "us-central1",
		ResourceType:      ResourceTypeStorage,
		Currency:          "USD",
		StorageGBPerMonth: 0.020, // Cloud Storage standard
	}

	db.Prices["gcp:us-central1:database"] = &Pricing{
		Provider:          ProviderGCP,
		Region:            "us-central1",
		ResourceType:      ResourceTypeDatabase,
		Currency:          "USD",
		CPUPerHour:        0.072,
		MemoryGBPerHour:   0.0143,
		StorageGBPerMonth: 0.17, // Cloud SQL
	}

	// Azure Pricing (eastus)
	db.Prices["azure:eastus:compute"] = &Pricing{
		Provider:          ProviderAzure,
		Region:            "eastus",
		ResourceType:      ResourceTypeCompute,
		Currency:          "USD",
		CPUPerHour:        0.051,
		MemoryGBPerHour:   0.0102,
		StorageGBPerMonth: 0.12, // Managed disk
		K8sClusterPerHour: 0.10, // AKS control plane
	}

	db.Prices["azure:eastus:storage"] = &Pricing{
		Provider:          ProviderAzure,
		Region:            "eastus",
		ResourceType:      ResourceTypeStorage,
		Currency:          "USD",
		StorageGBPerMonth: 0.024, // Blob storage
	}

	db.Prices["azure:eastus:database"] = &Pricing{
		Provider:          ProviderAzure,
		Region:            "eastus",
		ResourceType:      ResourceTypeDatabase,
		Currency:          "USD",
		CPUPerHour:        0.077,
		MemoryGBPerHour:   0.0154,
		StorageGBPerMonth: 0.125, // SQL Database
	}

	return db
}

// LoadPricingDB loads a pricing database from a file
func LoadPricingDB(filename string) (*PricingDB, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read pricing file: %w", err)
	}

	var db PricingDB
	if err := json.Unmarshal(data, &db); err != nil {
		return nil, fmt.Errorf("failed to parse pricing file: %w", err)
	}

	return &db, nil
}

// SavePricingDB saves a pricing database to a file
func (db *PricingDB) SavePricingDB(filename string) error {
	data, err := json.MarshalIndent(db, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal pricing database: %w", err)
	}

	return os.WriteFile(filename, data, 0644)
}

// GetPricing gets pricing for a specific provider/region/resource type
func (db *PricingDB) GetPricing(provider Provider, resourceType ResourceType, region string) (*Pricing, error) {
	key := fmt.Sprintf("%s:%s:%s", provider, region, resourceType)

	pricing, ok := db.Prices[key]
	if !ok {
		// Try default region
		key = fmt.Sprintf("%s:us-east-1:%s", provider, resourceType)
		pricing, ok = db.Prices[key]
		if !ok {
			return nil, fmt.Errorf("pricing not found for %s/%s/%s", provider, region, resourceType)
		}
	}

	return pricing, nil
}

// AddPricing adds pricing information to the database
func (db *PricingDB) AddPricing(pricing *Pricing) {
	key := fmt.Sprintf("%s:%s:%s", pricing.Provider, pricing.Region, pricing.ResourceType)
	db.Prices[key] = pricing
}
