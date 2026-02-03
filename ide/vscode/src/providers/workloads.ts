import * as vscode from 'vscode';
import { PlatformFoundryClient, Workload } from '../client';

export class WorkloadsProvider implements vscode.TreeDataProvider<WorkloadItem> {
    private _onDidChangeTreeData: vscode.EventEmitter<WorkloadItem | undefined | null | void> = new vscode.EventEmitter<WorkloadItem | undefined | null | void>();
    readonly onDidChangeTreeData: vscode.Event<WorkloadItem | undefined | null | void> = this._onDidChangeTreeData.event;

    constructor(private client: PlatformFoundryClient) {}

    refresh(): void {
        this._onDidChangeTreeData.fire();
    }

    getTreeItem(element: WorkloadItem): vscode.TreeItem {
        return element;
    }

    async getChildren(element?: WorkloadItem): Promise<WorkloadItem[]> {
        if (element) {
            return [];
        }

        try {
            const workloads = await this.client.getWorkloads();
            return workloads.map(w => new WorkloadItem(w));
        } catch (error) {
            return [];
        }
    }
}

export class WorkloadItem extends vscode.TreeItem {
    constructor(public readonly workload: Workload) {
        super(workload.name, vscode.TreeItemCollapsibleState.None);

        this.description = `${workload.readyReplicas}/${workload.replicas} - ${workload.environment}`;
        this.tooltip = new vscode.MarkdownString(
            `**${workload.name}**\n\n` +
            `- Environment: ${workload.environment}\n` +
            `- Status: ${workload.status}\n` +
            `- Replicas: ${workload.readyReplicas}/${workload.replicas}`
        );

        this.contextValue = 'workload';
        this.iconPath = this.getIcon();
    }

    get name(): string {
        return this.workload.name;
    }

    private getIcon(): vscode.ThemeIcon {
        if (this.workload.readyReplicas === this.workload.replicas && this.workload.replicas > 0) {
            return new vscode.ThemeIcon('check', new vscode.ThemeColor('testing.iconPassed'));
        } else if (this.workload.readyReplicas > 0) {
            return new vscode.ThemeIcon('warning', new vscode.ThemeColor('editorWarning.foreground'));
        } else if (this.workload.status === 'Running') {
            return new vscode.ThemeIcon('sync~spin');
        } else {
            return new vscode.ThemeIcon('circle-outline');
        }
    }
}
