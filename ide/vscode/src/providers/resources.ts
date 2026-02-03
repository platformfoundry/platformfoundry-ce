import * as vscode from 'vscode';
import { PlatformFoundryClient, Resource } from '../client';

export class ResourcesProvider implements vscode.TreeDataProvider<ResourceItem> {
    private _onDidChangeTreeData: vscode.EventEmitter<ResourceItem | undefined | null | void> = new vscode.EventEmitter<ResourceItem | undefined | null | void>();
    readonly onDidChangeTreeData: vscode.Event<ResourceItem | undefined | null | void> = this._onDidChangeTreeData.event;

    constructor(private client: PlatformFoundryClient) {}

    refresh(): void {
        this._onDidChangeTreeData.fire();
    }

    getTreeItem(element: ResourceItem): vscode.TreeItem {
        return element;
    }

    async getChildren(element?: ResourceItem): Promise<ResourceItem[]> {
        if (element) {
            return [];
        }

        try {
            const resources = await this.client.getResources();
            return resources.map(r => new ResourceItem(r));
        } catch (error) {
            return [];
        }
    }
}

export class ResourceItem extends vscode.TreeItem {
    constructor(public readonly resource: Resource) {
        super(resource.name, vscode.TreeItemCollapsibleState.None);

        this.description = `${resource.type} (${resource.provider})`;
        this.tooltip = new vscode.MarkdownString(
            `**${resource.name}**\n\n` +
            `- Type: ${resource.type}\n` +
            `- Provider: ${resource.provider}\n` +
            `- Status: ${resource.status}`
        );

        this.contextValue = 'resource';
        this.iconPath = this.getIcon();
    }

    private getIcon(): vscode.ThemeIcon {
        const typeIcons: Record<string, string> = {
            'postgres': 'database',
            'mysql': 'database',
            'redis': 'database',
            'mongodb': 'database',
            'sqs': 'mail',
            'sns': 'broadcast',
            's3': 'file',
            'dynamodb': 'table'
        };

        const iconName = typeIcons[this.resource.type] || 'package';

        if (this.resource.status === 'available') {
            return new vscode.ThemeIcon(iconName, new vscode.ThemeColor('testing.iconPassed'));
        } else if (this.resource.status === 'creating' || this.resource.status === 'updating') {
            return new vscode.ThemeIcon(iconName, new vscode.ThemeColor('editorWarning.foreground'));
        } else {
            return new vscode.ThemeIcon(iconName);
        }
    }
}
