import * as vscode from 'vscode';
import { PlatformFoundryClient, Deployment } from '../client';

export class DeploymentsProvider implements vscode.TreeDataProvider<DeploymentItem> {
    private _onDidChangeTreeData: vscode.EventEmitter<DeploymentItem | undefined | null | void> = new vscode.EventEmitter<DeploymentItem | undefined | null | void>();
    readonly onDidChangeTreeData: vscode.Event<DeploymentItem | undefined | null | void> = this._onDidChangeTreeData.event;

    constructor(private client: PlatformFoundryClient) {}

    refresh(): void {
        this._onDidChangeTreeData.fire();
    }

    getTreeItem(element: DeploymentItem): vscode.TreeItem {
        return element;
    }

    async getChildren(element?: DeploymentItem): Promise<DeploymentItem[]> {
        if (element) {
            return [];
        }

        try {
            const deployments = await this.client.getDeployments();
            return deployments.map(d => new DeploymentItem(d));
        } catch (error) {
            return [];
        }
    }
}

export class DeploymentItem extends vscode.TreeItem {
    constructor(public readonly deployment: Deployment) {
        super(`${deployment.workload} @ ${deployment.version}`, vscode.TreeItemCollapsibleState.None);

        this.description = `${deployment.environment} - ${deployment.status}`;
        this.tooltip = new vscode.MarkdownString(
            `**${deployment.workload}**\n\n` +
            `- ID: ${deployment.id}\n` +
            `- Version: ${deployment.version}\n` +
            `- Environment: ${deployment.environment}\n` +
            `- Status: ${deployment.status}\n` +
            `- Started: ${new Date(deployment.startedAt).toLocaleString()}`
        );

        this.contextValue = 'deployment';
        this.iconPath = this.getIcon();
    }

    private getIcon(): vscode.ThemeIcon {
        switch (this.deployment.status) {
            case 'succeeded':
                return new vscode.ThemeIcon('pass', new vscode.ThemeColor('testing.iconPassed'));
            case 'running':
                return new vscode.ThemeIcon('sync~spin');
            case 'failed':
                return new vscode.ThemeIcon('error', new vscode.ThemeColor('testing.iconFailed'));
            case 'cancelled':
                return new vscode.ThemeIcon('circle-slash');
            default:
                return new vscode.ThemeIcon('rocket');
        }
    }
}
