package aws

import (
	"context"
	"fmt"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/elasticache"
	ectypes "github.com/aws/aws-sdk-go-v2/service/elasticache/types"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	rdstypes "github.com/aws/aws-sdk-go-v2/service/rds/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/platformfoundry/pf-ce/pkg/plugin"
)

// Plugin implements the Plugin interface for AWS infrastructure
type Plugin struct {
	name    string
	version string

	region  string
	profile string

	ec2Client         *ec2.Client
	rdsClient         *rds.Client
	elasticacheClient *elasticache.Client
	s3Client          *s3.Client

	initialized bool
	mu          sync.RWMutex

	// Track created resources for deletion/status
	resources map[string]resourceInfo
}

type resourceInfo struct {
	Type string
	ID   string
	ARN  string
}

// Config represents the configuration schema for AWS resources
type Config struct {
	Region    string            `yaml:"region" json:"region"`
	Profile   string            `yaml:"profile,omitempty" json:"profile,omitempty"`
	Resources []Resource        `yaml:"resources" json:"resources"`
	Tags      map[string]string `yaml:"tags,omitempty" json:"tags,omitempty"`
}

// Resource represents an AWS resource to provision
type Resource struct {
	Type       string                 `yaml:"type" json:"type"`
	Name       string                 `yaml:"name" json:"name"`
	Properties map[string]interface{} `yaml:"properties" json:"properties"`
	DependsOn  []string               `yaml:"dependsOn,omitempty" json:"dependsOn,omitempty"`
}

// New creates a new AWS plugin
func New() *Plugin {
	return &Plugin{
		name:      "aws",
		version:   "1.0.0",
		resources: make(map[string]resourceInfo),
	}
}

func (p *Plugin) Name() string    { return p.name }
func (p *Plugin) Type() string    { return "Infrastructure" }
func (p *Plugin) Version() string { return p.version }

func (p *Plugin) ConfigType() interface{} {
	return Config{}
}

// Initialize sets up AWS clients
func (p *Plugin) Initialize(ctx context.Context, region, profile string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.initialized {
		return nil
	}

	p.region = region
	p.profile = profile

	opts := []func(*config.LoadOptions) error{
		config.WithRegion(region),
	}
	if profile != "" {
		opts = append(opts, config.WithSharedConfigProfile(profile))
	}

	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	p.ec2Client = ec2.NewFromConfig(cfg)
	p.rdsClient = rds.NewFromConfig(cfg)
	p.elasticacheClient = elasticache.NewFromConfig(cfg)
	p.s3Client = s3.NewFromConfig(cfg)
	p.initialized = true

	return nil
}

func (p *Plugin) Validate(spec map[string]interface{}) error {
	cfg, err := p.parseConfig(spec)
	if err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	validTypes := map[string]bool{
		"vpc": true, "subnet": true, "security_group": true,
		"rds": true, "elasticache": true, "s3": true,
		"internet_gateway": true, "nat_gateway": true, "route_table": true,
	}

	for _, res := range cfg.Resources {
		if !validTypes[res.Type] {
			return fmt.Errorf("unsupported resource type: %s", res.Type)
		}
		if res.Name == "" {
			return fmt.Errorf("resource name is required")
		}
	}

	return nil
}

func (p *Plugin) Plan(spec map[string]interface{}) (*plugin.Plan, error) {
	cfg, err := p.parseConfig(spec)
	if err != nil {
		return nil, err
	}

	plan := &plugin.Plan{
		Actions: make([]string, 0, len(cfg.Resources)),
		Changes: make(map[string]string),
	}

	for _, res := range cfg.Resources {
		action := fmt.Sprintf("create %s: %s", res.Type, res.Name)
		plan.Actions = append(plan.Actions, action)
		plan.Changes[res.Name] = res.Type
	}

	return plan, nil
}

func (p *Plugin) Apply(spec map[string]interface{}) (*plugin.Result, error) {
	cfg, err := p.parseConfig(spec)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	if !p.initialized {
		if err := p.Initialize(ctx, cfg.Region, cfg.Profile); err != nil {
			return nil, err
		}
	}

	result := &plugin.Result{
		Status:    "success",
		Resources: make([]string, 0),
		Outputs:   make(map[string]string),
	}

	// Sort resources by dependencies
	ordered, err := p.topologicalSort(cfg.Resources)
	if err != nil {
		return nil, fmt.Errorf("dependency resolution failed: %w", err)
	}

	// Apply resources in order
	for _, res := range ordered {
		output, err := p.applyResource(ctx, res, cfg.Tags)
		if err != nil {
			result.Status = "partial"
			result.Message = fmt.Sprintf("failed at %s/%s: %v", res.Type, res.Name, err)
			return result, err
		}

		result.Resources = append(result.Resources, fmt.Sprintf("%s:%s", res.Type, res.Name))
		for k, v := range output {
			result.Outputs[fmt.Sprintf("%s.%s", res.Name, k)] = v
		}
	}

	result.Message = fmt.Sprintf("Created %d resources", len(result.Resources))
	return result, nil
}

func (p *Plugin) Delete(name string) error {
	p.mu.RLock()
	info, exists := p.resources[name]
	p.mu.RUnlock()

	if !exists {
		return fmt.Errorf("resource %s not found", name)
	}

	ctx := context.Background()

	switch info.Type {
	case "vpc":
		_, err := p.ec2Client.DeleteVpc(ctx, &ec2.DeleteVpcInput{
			VpcId: aws.String(info.ID),
		})
		return err
	case "subnet":
		_, err := p.ec2Client.DeleteSubnet(ctx, &ec2.DeleteSubnetInput{
			SubnetId: aws.String(info.ID),
		})
		return err
	case "security_group":
		_, err := p.ec2Client.DeleteSecurityGroup(ctx, &ec2.DeleteSecurityGroupInput{
			GroupId: aws.String(info.ID),
		})
		return err
	case "rds":
		_, err := p.rdsClient.DeleteDBInstance(ctx, &rds.DeleteDBInstanceInput{
			DBInstanceIdentifier: aws.String(info.ID),
			SkipFinalSnapshot:    aws.Bool(true),
		})
		return err
	case "elasticache":
		_, err := p.elasticacheClient.DeleteCacheCluster(ctx, &elasticache.DeleteCacheClusterInput{
			CacheClusterId: aws.String(info.ID),
		})
		return err
	case "s3":
		_, err := p.s3Client.DeleteBucket(ctx, &s3.DeleteBucketInput{
			Bucket: aws.String(info.ID),
		})
		return err
	default:
		return fmt.Errorf("unsupported resource type for deletion: %s", info.Type)
	}
}

func (p *Plugin) Status(name string) (*plugin.Status, error) {
	p.mu.RLock()
	info, exists := p.resources[name]
	p.mu.RUnlock()

	if !exists {
		return &plugin.Status{
			State:   "unknown",
			Ready:   false,
			Message: "Resource not found in local state",
		}, nil
	}

	ctx := context.Background()

	switch info.Type {
	case "rds":
		out, err := p.rdsClient.DescribeDBInstances(ctx, &rds.DescribeDBInstancesInput{
			DBInstanceIdentifier: aws.String(info.ID),
		})
		if err != nil {
			return nil, err
		}
		if len(out.DBInstances) == 0 {
			return &plugin.Status{State: "deleted", Ready: false}, nil
		}
		status := aws.ToString(out.DBInstances[0].DBInstanceStatus)
		return &plugin.Status{
			State:   status,
			Ready:   status == "available",
			Message: fmt.Sprintf("RDS instance is %s", status),
			Details: map[string]string{
				"endpoint": aws.ToString(out.DBInstances[0].Endpoint.Address),
			},
		}, nil
	default:
		return &plugin.Status{
			State:   "unknown",
			Ready:   true,
			Message: "Status check not implemented for this resource type",
		}, nil
	}
}

func (p *Plugin) applyResource(ctx context.Context, res Resource, tags map[string]string) (map[string]string, error) {
	switch res.Type {
	case "vpc":
		return p.createVPC(ctx, res, tags)
	case "subnet":
		return p.createSubnet(ctx, res, tags)
	case "security_group":
		return p.createSecurityGroup(ctx, res, tags)
	case "rds":
		return p.createRDS(ctx, res, tags)
	case "elasticache":
		return p.createElastiCache(ctx, res, tags)
	case "s3":
		return p.createS3Bucket(ctx, res, tags)
	default:
		return nil, fmt.Errorf("unsupported resource type: %s", res.Type)
	}
}

func (p *Plugin) createVPC(ctx context.Context, res Resource, tags map[string]string) (map[string]string, error) {
	cidr := getStringProp(res.Properties, "cidr_block", "10.0.0.0/16")

	out, err := p.ec2Client.CreateVpc(ctx, &ec2.CreateVpcInput{
		CidrBlock: aws.String(cidr),
		TagSpecifications: []ec2types.TagSpecification{
			{
				ResourceType: ec2types.ResourceTypeVpc,
				Tags:         p.buildTags(res.Name, tags),
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create VPC: %w", err)
	}

	vpcID := aws.ToString(out.Vpc.VpcId)
	p.trackResource(res.Name, "vpc", vpcID, "")

	// Enable DNS hostnames
	_, _ = p.ec2Client.ModifyVpcAttribute(ctx, &ec2.ModifyVpcAttributeInput{
		VpcId:              aws.String(vpcID),
		EnableDnsHostnames: &ec2types.AttributeBooleanValue{Value: aws.Bool(true)},
	})

	return map[string]string{
		"id":   vpcID,
		"cidr": cidr,
	}, nil
}

func (p *Plugin) createSubnet(ctx context.Context, res Resource, tags map[string]string) (map[string]string, error) {
	vpcID := getStringProp(res.Properties, "vpc_id", "")
	cidr := getStringProp(res.Properties, "cidr_block", "")
	az := getStringProp(res.Properties, "availability_zone", "")

	if vpcID == "" || cidr == "" {
		return nil, fmt.Errorf("vpc_id and cidr_block are required for subnet")
	}

	input := &ec2.CreateSubnetInput{
		VpcId:     aws.String(vpcID),
		CidrBlock: aws.String(cidr),
		TagSpecifications: []ec2types.TagSpecification{
			{
				ResourceType: ec2types.ResourceTypeSubnet,
				Tags:         p.buildTags(res.Name, tags),
			},
		},
	}
	if az != "" {
		input.AvailabilityZone = aws.String(az)
	}

	out, err := p.ec2Client.CreateSubnet(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to create subnet: %w", err)
	}

	subnetID := aws.ToString(out.Subnet.SubnetId)
	p.trackResource(res.Name, "subnet", subnetID, "")

	return map[string]string{
		"id":   subnetID,
		"cidr": cidr,
	}, nil
}

func (p *Plugin) createSecurityGroup(ctx context.Context, res Resource, tags map[string]string) (map[string]string, error) {
	vpcID := getStringProp(res.Properties, "vpc_id", "")
	description := getStringProp(res.Properties, "description", "Managed by PlatformFoundry")

	out, err := p.ec2Client.CreateSecurityGroup(ctx, &ec2.CreateSecurityGroupInput{
		GroupName:   aws.String(res.Name),
		Description: aws.String(description),
		VpcId:       aws.String(vpcID),
		TagSpecifications: []ec2types.TagSpecification{
			{
				ResourceType: ec2types.ResourceTypeSecurityGroup,
				Tags:         p.buildTags(res.Name, tags),
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create security group: %w", err)
	}

	sgID := aws.ToString(out.GroupId)
	p.trackResource(res.Name, "security_group", sgID, "")

	// Add ingress rules if specified
	if rules, ok := res.Properties["ingress"].([]interface{}); ok {
		for _, r := range rules {
			if rule, ok := r.(map[string]interface{}); ok {
				_, _ = p.ec2Client.AuthorizeSecurityGroupIngress(ctx, &ec2.AuthorizeSecurityGroupIngressInput{
					GroupId: aws.String(sgID),
					IpPermissions: []ec2types.IpPermission{
						{
							IpProtocol: aws.String(getStringProp(rule, "protocol", "tcp")),
							FromPort:   aws.Int32(int32(getIntProp(rule, "from_port", 0))),
							ToPort:     aws.Int32(int32(getIntProp(rule, "to_port", 0))),
							IpRanges: []ec2types.IpRange{
								{CidrIp: aws.String(getStringProp(rule, "cidr", "0.0.0.0/0"))},
							},
						},
					},
				})
			}
		}
	}

	return map[string]string{
		"id": sgID,
	}, nil
}

func (p *Plugin) createRDS(ctx context.Context, res Resource, tags map[string]string) (map[string]string, error) {
	engine := getStringProp(res.Properties, "engine", "postgres")
	engineVersion := getStringProp(res.Properties, "engine_version", "15")
	instanceClass := getStringProp(res.Properties, "instance_class", "db.t3.micro")
	allocatedStorage := int32(getIntProp(res.Properties, "allocated_storage", 20))
	masterUsername := getStringProp(res.Properties, "master_username", "admin")
	masterPassword := getStringProp(res.Properties, "master_password", "")
	dbName := getStringProp(res.Properties, "db_name", "app")

	if masterPassword == "" {
		return nil, fmt.Errorf("master_password is required for RDS")
	}

	rdsTags := make([]rdstypes.Tag, 0)
	for k, v := range tags {
		rdsTags = append(rdsTags, rdstypes.Tag{Key: aws.String(k), Value: aws.String(v)})
	}
	rdsTags = append(rdsTags, rdstypes.Tag{Key: aws.String("Name"), Value: aws.String(res.Name)})

	out, err := p.rdsClient.CreateDBInstance(ctx, &rds.CreateDBInstanceInput{
		DBInstanceIdentifier: aws.String(res.Name),
		DBInstanceClass:      aws.String(instanceClass),
		Engine:               aws.String(engine),
		EngineVersion:        aws.String(engineVersion),
		AllocatedStorage:     aws.Int32(allocatedStorage),
		MasterUsername:       aws.String(masterUsername),
		MasterUserPassword:   aws.String(masterPassword),
		DBName:               aws.String(dbName),
		PubliclyAccessible:   aws.Bool(false),
		Tags:                 rdsTags,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create RDS instance: %w", err)
	}

	dbID := aws.ToString(out.DBInstance.DBInstanceIdentifier)
	dbARN := aws.ToString(out.DBInstance.DBInstanceArn)
	p.trackResource(res.Name, "rds", dbID, dbARN)

	return map[string]string{
		"id":       dbID,
		"arn":      dbARN,
		"engine":   engine,
		"endpoint": "pending", // Endpoint available after instance is ready
	}, nil
}

func (p *Plugin) createElastiCache(ctx context.Context, res Resource, tags map[string]string) (map[string]string, error) {
	engine := getStringProp(res.Properties, "engine", "redis")
	nodeType := getStringProp(res.Properties, "node_type", "cache.t3.micro")
	numNodes := int32(getIntProp(res.Properties, "num_cache_nodes", 1))
	port := int32(getIntProp(res.Properties, "port", 6379))

	ecTags := make([]ectypes.Tag, 0)
	for k, v := range tags {
		ecTags = append(ecTags, ectypes.Tag{Key: aws.String(k), Value: aws.String(v)})
	}
	ecTags = append(ecTags, ectypes.Tag{Key: aws.String("Name"), Value: aws.String(res.Name)})

	out, err := p.elasticacheClient.CreateCacheCluster(ctx, &elasticache.CreateCacheClusterInput{
		CacheClusterId:   aws.String(res.Name),
		Engine:           aws.String(engine),
		CacheNodeType:    aws.String(nodeType),
		NumCacheNodes:    aws.Int32(numNodes),
		Port:             aws.Int32(port),
		Tags:             ecTags,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create ElastiCache cluster: %w", err)
	}

	clusterID := aws.ToString(out.CacheCluster.CacheClusterId)
	p.trackResource(res.Name, "elasticache", clusterID, "")

	return map[string]string{
		"id":     clusterID,
		"engine": engine,
		"port":   fmt.Sprintf("%d", port),
	}, nil
}

func (p *Plugin) createS3Bucket(ctx context.Context, res Resource, tags map[string]string) (map[string]string, error) {
	bucketName := getStringProp(res.Properties, "bucket", res.Name)

	input := &s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
	}

	// Add location constraint for non-us-east-1 regions
	if p.region != "us-east-1" {
		input.CreateBucketConfiguration = &s3types.CreateBucketConfiguration{
			LocationConstraint: s3types.BucketLocationConstraint(p.region),
		}
	}

	_, err := p.s3Client.CreateBucket(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to create S3 bucket: %w", err)
	}

	p.trackResource(res.Name, "s3", bucketName, "")

	// Enable encryption
	_, _ = p.s3Client.PutBucketEncryption(ctx, &s3.PutBucketEncryptionInput{
		Bucket: aws.String(bucketName),
		ServerSideEncryptionConfiguration: &s3types.ServerSideEncryptionConfiguration{
			Rules: []s3types.ServerSideEncryptionRule{
				{
					ApplyServerSideEncryptionByDefault: &s3types.ServerSideEncryptionByDefault{
						SSEAlgorithm: s3types.ServerSideEncryptionAes256,
					},
				},
			},
		},
	})

	return map[string]string{
		"id":     bucketName,
		"arn":    fmt.Sprintf("arn:aws:s3:::%s", bucketName),
		"region": p.region,
	}, nil
}

func (p *Plugin) trackResource(name, resType, id, arn string) {
	p.mu.Lock()
	p.resources[name] = resourceInfo{Type: resType, ID: id, ARN: arn}
	p.mu.Unlock()
}

func (p *Plugin) buildTags(name string, tags map[string]string) []ec2types.Tag {
	result := make([]ec2types.Tag, 0, len(tags)+2)
	result = append(result, ec2types.Tag{Key: aws.String("Name"), Value: aws.String(name)})
	result = append(result, ec2types.Tag{Key: aws.String("managed-by"), Value: aws.String("platformfoundry")})
	for k, v := range tags {
		result = append(result, ec2types.Tag{Key: aws.String(k), Value: aws.String(v)})
	}
	return result
}

func (p *Plugin) parseConfig(spec map[string]interface{}) (*Config, error) {
	cfg := &Config{
		Region:    "us-east-1",
		Resources: make([]Resource, 0),
		Tags:      make(map[string]string),
	}

	if r, ok := spec["region"].(string); ok {
		cfg.Region = r
	}
	if pr, ok := spec["profile"].(string); ok {
		cfg.Profile = pr
	}
	if t, ok := spec["tags"].(map[string]interface{}); ok {
		for k, v := range t {
			if vs, ok := v.(string); ok {
				cfg.Tags[k] = vs
			}
		}
	}

	if resources, ok := spec["resources"].([]interface{}); ok {
		for _, r := range resources {
			if rm, ok := r.(map[string]interface{}); ok {
				res := Resource{
					Type:       getStringProp(rm, "type", ""),
					Name:       getStringProp(rm, "name", ""),
					Properties: make(map[string]interface{}),
				}
				if props, ok := rm["properties"].(map[string]interface{}); ok {
					res.Properties = props
				}
				if deps, ok := rm["dependsOn"].([]interface{}); ok {
					for _, d := range deps {
						if ds, ok := d.(string); ok {
							res.DependsOn = append(res.DependsOn, ds)
						}
					}
				}
				cfg.Resources = append(cfg.Resources, res)
			}
		}
	}

	return cfg, nil
}

func (p *Plugin) topologicalSort(resources []Resource) ([]Resource, error) {
	// Build adjacency list
	graph := make(map[string][]string)
	inDegree := make(map[string]int)
	resMap := make(map[string]Resource)

	for _, res := range resources {
		graph[res.Name] = res.DependsOn
		inDegree[res.Name] = len(res.DependsOn)
		resMap[res.Name] = res
	}

	// Find nodes with no dependencies
	var queue []string
	for name, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, name)
		}
	}

	var sorted []Resource
	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]
		sorted = append(sorted, resMap[name])

		// Update dependents
		for depName, deps := range graph {
			for _, d := range deps {
				if d == name {
					inDegree[depName]--
					if inDegree[depName] == 0 {
						queue = append(queue, depName)
					}
				}
			}
		}
	}

	if len(sorted) != len(resources) {
		return nil, fmt.Errorf("circular dependency detected")
	}

	return sorted, nil
}

func getStringProp(m map[string]interface{}, key, defaultVal string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return defaultVal
}

func getIntProp(m map[string]interface{}, key string, defaultVal int) int {
	if v, ok := m[key].(int); ok {
		return v
	}
	if v, ok := m[key].(float64); ok {
		return int(v)
	}
	return defaultVal
}
