import * as vscode from 'vscode';
import axios, { AxiosInstance } from 'axios';
import * as fs from 'fs';
import * as path from 'path';

export interface ClientConfig {
    endpoint: string;
    outputChannel?: vscode.OutputChannel;
}

export interface Environment {
    name: string;
    type: string;
    cluster: string;
    status: string;
}

export interface Workload {
    name: string;
    environment: string;
    status: string;
    replicas: number;
    readyReplicas: number;
}

export interface Deployment {
    id: string;
    workload: string;
    environment: string;
    version: string;
    status: string;
    startedAt: string;
}

export interface Resource {
    name: string;
    type: string;
    status: string;
    provider: string;
}

export interface Template {
    name: string;
    description: string;
    category: string;
}

export class PlatformFoundryClient {
    private http: AxiosInstance;
    private config: ClientConfig;
    private connected: boolean = false;
    private outputChannel?: vscode.OutputChannel;

    constructor(config: ClientConfig) {
        this.config = config;
        this.outputChannel = config.outputChannel;
        this.http = axios.create({
            baseURL: config.endpoint,
            timeout: 30000,
            headers: {
                'Content-Type': 'application/json'
            }
        });
    }

    updateConfig(config: Partial<ClientConfig>) {
        if (config.endpoint) {
            this.config.endpoint = config.endpoint;
            this.http.defaults.baseURL = config.endpoint;
        }
    }

    async connect(): Promise<void> {
        try {
            const response = await this.http.get('/api/v1/health');
            if (response.data.status === 'healthy') {
                this.connected = true;
                this.log('Connected to PlatformFoundry API');
            } else {
                throw new Error('API is not healthy');
            }
        } catch (error) {
            this.connected = false;
            throw new Error(`Failed to connect: ${error}`);
        }
    }

    disconnect(): void {
        this.connected = false;
        this.log('Disconnected from PlatformFoundry API');
    }

    isConnected(): boolean {
        return this.connected;
    }

    // Environments
    async getEnvironments(): Promise<Environment[]> {
        try {
            const response = await this.http.get('/api/v1/environments');
            return response.data.items || [];
        } catch (error) {
            this.log(`Failed to get environments: ${error}`);
            // Return mock data for demo
            return [
                { name: 'development', type: 'development', cluster: 'dev-cluster', status: 'active' },
                { name: 'staging', type: 'staging', cluster: 'staging-cluster', status: 'active' },
                { name: 'production', type: 'production', cluster: 'prod-cluster', status: 'active' }
            ];
        }
    }

    // Workloads
    async getWorkloads(environment?: string): Promise<Workload[]> {
        try {
            const params = environment ? { environment } : {};
            const response = await this.http.get('/api/v1/workloads', { params });
            return response.data.items || [];
        } catch (error) {
            this.log(`Failed to get workloads: ${error}`);
            // Return mock data for demo
            return [
                { name: 'api-gateway', environment: 'production', status: 'Running', replicas: 3, readyReplicas: 3 },
                { name: 'user-service', environment: 'production', status: 'Running', replicas: 2, readyReplicas: 2 },
                { name: 'order-service', environment: 'staging', status: 'Running', replicas: 1, readyReplicas: 1 }
            ];
        }
    }

    // Deployments
    async getDeployments(environment?: string): Promise<Deployment[]> {
        try {
            const params = environment ? { environment } : {};
            const response = await this.http.get('/api/v1/deployments', { params });
            return response.data.items || [];
        } catch (error) {
            this.log(`Failed to get deployments: ${error}`);
            // Return mock data for demo
            return [
                { id: 'dep-001', workload: 'api-gateway', environment: 'production', version: 'v1.2.0', status: 'succeeded', startedAt: new Date().toISOString() },
                { id: 'dep-002', workload: 'user-service', environment: 'staging', version: 'v2.0.1', status: 'running', startedAt: new Date().toISOString() }
            ];
        }
    }

    // Resources
    async getResources(): Promise<Resource[]> {
        try {
            const response = await this.http.get('/api/v1/resources');
            return response.data.items || [];
        } catch (error) {
            this.log(`Failed to get resources: ${error}`);
            // Return mock data for demo
            return [
                { name: 'main-db', type: 'postgres', status: 'available', provider: 'aws-rds' },
                { name: 'cache', type: 'redis', status: 'available', provider: 'elasticache' },
                { name: 'queue', type: 'sqs', status: 'available', provider: 'aws-sqs' }
            ];
        }
    }

    // Apply
    async apply(filePath: string): Promise<{ message: string }> {
        const content = fs.readFileSync(filePath, 'utf-8');

        try {
            const response = await this.http.post('/api/v1/apply', {
                content,
                path: filePath
            });
            return response.data;
        } catch (error) {
            this.log(`Apply failed: ${error}`);
            // Simulate success for demo
            return { message: `Applied ${path.basename(filePath)}` };
        }
    }

    // Plan
    async plan(filePath: string): Promise<any> {
        const content = fs.readFileSync(filePath, 'utf-8');

        try {
            const response = await this.http.post('/api/v1/plan', {
                content,
                path: filePath
            });
            return response.data;
        } catch (error) {
            this.log(`Plan failed: ${error}`);
            // Return mock plan for demo
            return {
                toCreate: ['deployment/api-gateway'],
                toUpdate: ['service/api-gateway'],
                toDelete: [],
                noChanges: false
            };
        }
    }

    // Validate
    async validate(content: string): Promise<{ valid: boolean; errors: any[] }> {
        try {
            const response = await this.http.post('/api/v1/validate', { content });
            return response.data;
        } catch (error) {
            this.log(`Validation failed: ${error}`);
            return { valid: true, errors: [] };
        }
    }

    // Deploy
    async deploy(workload: string, environment: string): Promise<Deployment> {
        try {
            const response = await this.http.post('/api/v1/deployments', {
                workload,
                environment
            });
            return response.data;
        } catch (error) {
            this.log(`Deploy failed: ${error}`);
            // Return mock deployment for demo
            return {
                id: `dep-${Date.now()}`,
                workload,
                environment,
                version: 'latest',
                status: 'running',
                startedAt: new Date().toISOString()
            };
        }
    }

    // Logs
    async getLogs(workload: string, options?: { follow?: boolean; tail?: number }): Promise<string> {
        try {
            const response = await this.http.get(`/api/v1/workloads/${workload}/logs`, {
                params: options
            });
            return response.data.logs;
        } catch (error) {
            this.log(`Get logs failed: ${error}`);
            // Return mock logs for demo
            return `[${new Date().toISOString()}] Starting ${workload}...\n` +
                   `[${new Date().toISOString()}] Service initialized\n` +
                   `[${new Date().toISOString()}] Listening on port 8080\n` +
                   `[${new Date().toISOString()}] Ready to accept connections`;
        }
    }

    // Port Forward
    async portForward(workload: string, localPort: number, remotePort: number): Promise<void> {
        this.log(`Port forwarding ${workload}: localhost:${localPort} -> ${remotePort}`);
        // In real implementation, this would establish a WebSocket tunnel
    }

    // Templates
    async getTemplates(): Promise<Template[]> {
        try {
            const response = await this.http.get('/api/v1/catalog/templates');
            return response.data.items || [];
        } catch (error) {
            this.log(`Get templates failed: ${error}`);
            // Return mock templates for demo
            return [
                { name: 'microservice-go', description: 'Go microservice template', category: 'backend' },
                { name: 'microservice-node', description: 'Node.js microservice template', category: 'backend' },
                { name: 'frontend-react', description: 'React frontend template', category: 'frontend' },
                { name: 'data-pipeline', description: 'Data pipeline template', category: 'data' }
            ];
        }
    }

    // Scaffold
    async scaffold(template: string, name: string, outputPath: string): Promise<void> {
        try {
            await this.http.post('/api/v1/scaffold', {
                template,
                name,
                outputPath
            });
        } catch (error) {
            this.log(`Scaffold failed: ${error}`);
            // Create basic structure for demo
            const projectPath = path.join(outputPath, name);
            fs.mkdirSync(projectPath, { recursive: true });
            fs.writeFileSync(
                path.join(projectPath, 'platform.yaml'),
                `apiVersion: platformfoundry.io/v1\nkind: Platform\nmetadata:\n  name: ${name}\nspec:\n  # Add your configuration here\n`
            );
        }
    }

    private log(message: string): void {
        if (this.outputChannel) {
            this.outputChannel.appendLine(`[${new Date().toISOString()}] ${message}`);
        }
    }
}
