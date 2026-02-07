import * as vscode from 'vscode';
import { PlatformFoundryClient, Environment } from '../client';

export class EnvironmentsProvider implements vscode.TreeDataProvider<EnvironmentItem> {
    private _onDidChangeTreeData: vscode.EventEmitter<EnvironmentItem | undefined | null | void> = new vscode.EventEmitter<EnvironmentItem | undefined | null | void>();
    readonly onDidChangeTreeData: vscode.Event<EnvironmentItem | undefined | null | void> = this._onDidChangeTreeData.event;

    constructor(private client: PlatformFoundryClient) {}

    refresh(): void {
        this._onDidChangeTreeData.fire();
    }

    getTreeItem(element: EnvironmentItem): vscode.TreeItem {
        return element;
    }

    async getChildren(element?: EnvironmentItem): Promise<EnvironmentItem[]> {
        if (element) {
            return [];
        }

        try {
            const environments = await this.client.getEnvironments();
            return environments.map(env => new EnvironmentItem(env));
        } catch (error) {
            return [];
        }
    }
}

export class EnvironmentItem extends vscode.TreeItem {
    constructor(public readonly environment: Environment) {
        super(environment.name, vscode.TreeItemCollapsibleState.None);

        this.description = `${environment.type} - ${environment.cluster}`;
        this.tooltip = new vscode.MarkdownString(
            `**${environment.name}**\n\n` +
            `- Type: ${environment.type}\n` +
            `- Cluster: ${environment.cluster}\n` +
            `- Status: ${environment.status}`
        );

        this.contextValue = 'environment';
        this.iconPath = this.getIcon();
    }

    private getIcon(): vscode.ThemeIcon {
        switch (this.environment.status) {
            case 'active':
                return new vscode.ThemeIcon('pass', new vscode.ThemeColor('testing.iconPassed'));
            case 'inactive':
                return new vscode.ThemeIcon('circle-outline');
            case 'error':
                return new vscode.ThemeIcon('error', new vscode.ThemeColor('testing.iconFailed'));
            default:
                return new vscode.ThemeIcon('globe');
        }
    }
}
