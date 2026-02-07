import * as vscode from 'vscode';
import { PlatformFoundryClient } from './client';
import { EnvironmentsProvider } from './providers/environments';
import { WorkloadsProvider } from './providers/workloads';
import { DeploymentsProvider } from './providers/deployments';
import { ResourcesProvider } from './providers/resources';
import { LogsPanel } from './panels/logs';
import { validateDocument } from './validation';

let client: PlatformFoundryClient;
let statusBarItem: vscode.StatusBarItem;
let outputChannel: vscode.OutputChannel;

export function activate(context: vscode.ExtensionContext) {
    outputChannel = vscode.window.createOutputChannel('PlatformFoundry');
    outputChannel.appendLine('PlatformFoundry extension activated');

    // Initialize client
    const config = vscode.workspace.getConfiguration('platformfoundry');
    client = new PlatformFoundryClient({
        endpoint: config.get('apiEndpoint') || 'http://localhost:8080',
        outputChannel
    });

    // Create status bar item
    statusBarItem = vscode.window.createStatusBarItem(vscode.StatusBarAlignment.Left, 100);
    statusBarItem.command = 'platformfoundry.connect';
    updateStatusBar('disconnected');
    if (config.get('showStatusBar')) {
        statusBarItem.show();
    }

    // Register tree data providers
    const environmentsProvider = new EnvironmentsProvider(client);
    const workloadsProvider = new WorkloadsProvider(client);
    const deploymentsProvider = new DeploymentsProvider(client);
    const resourcesProvider = new ResourcesProvider(client);

    vscode.window.registerTreeDataProvider('platformfoundry.environments', environmentsProvider);
    vscode.window.registerTreeDataProvider('platformfoundry.workloads', workloadsProvider);
    vscode.window.registerTreeDataProvider('platformfoundry.deployments', deploymentsProvider);
    vscode.window.registerTreeDataProvider('platformfoundry.resources', resourcesProvider);

    // Register commands
    context.subscriptions.push(
        vscode.commands.registerCommand('platformfoundry.connect', async () => {
            try {
                await client.connect();
                updateStatusBar('connected');
                vscode.window.showInformationMessage('Connected to PlatformFoundry');
                refreshAllProviders();
            } catch (error) {
                vscode.window.showErrorMessage(`Failed to connect: ${error}`);
            }
        }),

        vscode.commands.registerCommand('platformfoundry.disconnect', () => {
            client.disconnect();
            updateStatusBar('disconnected');
            vscode.window.showInformationMessage('Disconnected from PlatformFoundry');
        }),

        vscode.commands.registerCommand('platformfoundry.apply', async () => {
            const editor = vscode.window.activeTextEditor;
            if (!editor) {
                vscode.window.showErrorMessage('No active editor');
                return;
            }

            const document = editor.document;
            await document.save();

            try {
                const result = await client.apply(document.uri.fsPath);
                vscode.window.showInformationMessage(`Apply successful: ${result.message}`);
                refreshAllProviders();
            } catch (error) {
                vscode.window.showErrorMessage(`Apply failed: ${error}`);
            }
        }),

        vscode.commands.registerCommand('platformfoundry.plan', async () => {
            const editor = vscode.window.activeTextEditor;
            if (!editor) {
                vscode.window.showErrorMessage('No active editor');
                return;
            }

            try {
                const result = await client.plan(editor.document.uri.fsPath);
                showPlanResult(result);
            } catch (error) {
                vscode.window.showErrorMessage(`Plan failed: ${error}`);
            }
        }),

        vscode.commands.registerCommand('platformfoundry.validate', async () => {
            const editor = vscode.window.activeTextEditor;
            if (!editor) {
                vscode.window.showErrorMessage('No active editor');
                return;
            }

            const diagnostics = await validateDocument(editor.document, client);
            if (diagnostics.length === 0) {
                vscode.window.showInformationMessage('Validation passed');
            } else {
                vscode.window.showWarningMessage(`Found ${diagnostics.length} issues`);
            }
        }),

        vscode.commands.registerCommand('platformfoundry.deploy', async (item?: any) => {
            let workloadName: string | undefined;

            if (item && item.name) {
                workloadName = item.name;
            } else {
                workloadName = await vscode.window.showInputBox({
                    prompt: 'Enter workload name',
                    placeHolder: 'my-workload'
                });
            }

            if (!workloadName) return;

            const environment = await vscode.window.showQuickPick(
                ['development', 'staging', 'production'],
                { placeHolder: 'Select environment' }
            );

            if (!environment) return;

            try {
                await vscode.window.withProgress({
                    location: vscode.ProgressLocation.Notification,
                    title: `Deploying ${workloadName} to ${environment}`,
                    cancellable: true
                }, async (progress, token) => {
                    const result = await client.deploy(workloadName!, environment);
                    return result;
                });

                vscode.window.showInformationMessage(`Deployment started for ${workloadName}`);
                deploymentsProvider.refresh();
            } catch (error) {
                vscode.window.showErrorMessage(`Deployment failed: ${error}`);
            }
        }),

        vscode.commands.registerCommand('platformfoundry.logs', async (item?: any) => {
            let workloadName: string | undefined;

            if (item && item.name) {
                workloadName = item.name;
            } else {
                workloadName = await vscode.window.showInputBox({
                    prompt: 'Enter workload name',
                    placeHolder: 'my-workload'
                });
            }

            if (!workloadName) return;

            LogsPanel.createOrShow(context.extensionUri, client, workloadName);
        }),

        vscode.commands.registerCommand('platformfoundry.portForward', async (item?: any) => {
            let workloadName: string | undefined;

            if (item && item.name) {
                workloadName = item.name;
            } else {
                workloadName = await vscode.window.showInputBox({
                    prompt: 'Enter workload name'
                });
            }

            if (!workloadName) return;

            const localPort = await vscode.window.showInputBox({
                prompt: 'Enter local port',
                placeHolder: '8080'
            });

            const remotePort = await vscode.window.showInputBox({
                prompt: 'Enter remote port',
                placeHolder: '80'
            });

            if (!localPort || !remotePort) return;

            try {
                await client.portForward(workloadName, parseInt(localPort), parseInt(remotePort));
                vscode.window.showInformationMessage(
                    `Port forwarding: localhost:${localPort} -> ${workloadName}:${remotePort}`
                );
            } catch (error) {
                vscode.window.showErrorMessage(`Port forward failed: ${error}`);
            }
        }),

        vscode.commands.registerCommand('platformfoundry.openTerminal', async (item?: any) => {
            let workloadName: string | undefined;

            if (item && item.name) {
                workloadName = item.name;
            } else {
                workloadName = await vscode.window.showInputBox({
                    prompt: 'Enter workload name'
                });
            }

            if (!workloadName) return;

            const terminal = vscode.window.createTerminal({
                name: `PF: ${workloadName}`,
                shellPath: 'pf',
                shellArgs: ['exec', '-it', workloadName, '--', '/bin/sh']
            });
            terminal.show();
        }),

        vscode.commands.registerCommand('platformfoundry.refreshExplorer', () => {
            refreshAllProviders();
        }),

        vscode.commands.registerCommand('platformfoundry.scaffold', async () => {
            const templates = await client.getTemplates();
            const selected = await vscode.window.showQuickPick(
                templates.map(t => ({ label: t.name, description: t.description })),
                { placeHolder: 'Select a template' }
            );

            if (!selected) return;

            const name = await vscode.window.showInputBox({
                prompt: 'Enter project name',
                placeHolder: 'my-project'
            });

            if (!name) return;

            const folder = await vscode.window.showOpenDialog({
                canSelectFolders: true,
                canSelectFiles: false,
                canSelectMany: false,
                openLabel: 'Select folder'
            });

            if (!folder || folder.length === 0) return;

            try {
                await client.scaffold(selected.label, name, folder[0].fsPath);
                vscode.window.showInformationMessage(`Project ${name} created successfully`);

                // Open the new folder
                const newFolder = vscode.Uri.joinPath(folder[0], name);
                vscode.commands.executeCommand('vscode.openFolder', newFolder);
            } catch (error) {
                vscode.window.showErrorMessage(`Scaffold failed: ${error}`);
            }
        }),

        vscode.commands.registerCommand('platformfoundry.openDocs', () => {
            vscode.env.openExternal(vscode.Uri.parse('https://docs.platformfoundry.io'));
        })
    );

    // Auto-validate on save
    if (config.get('autoValidate')) {
        context.subscriptions.push(
            vscode.workspace.onDidSaveTextDocument(async (document) => {
                if (isPlatformFoundryFile(document)) {
                    await validateDocument(document, client);
                }
            })
        );
    }

    // Configuration change handler
    context.subscriptions.push(
        vscode.workspace.onDidChangeConfiguration((event) => {
            if (event.affectsConfiguration('platformfoundry')) {
                const newConfig = vscode.workspace.getConfiguration('platformfoundry');
                client.updateConfig({
                    endpoint: newConfig.get('apiEndpoint') || 'http://localhost:8080'
                });

                if (newConfig.get('showStatusBar')) {
                    statusBarItem.show();
                } else {
                    statusBarItem.hide();
                }
            }
        })
    );

    // Disposables
    context.subscriptions.push(statusBarItem);
    context.subscriptions.push(outputChannel);

    // Helper functions
    function refreshAllProviders() {
        environmentsProvider.refresh();
        workloadsProvider.refresh();
        deploymentsProvider.refresh();
        resourcesProvider.refresh();
    }

    function updateStatusBar(status: 'connected' | 'disconnected' | 'error') {
        switch (status) {
            case 'connected':
                statusBarItem.text = '$(cloud) PF: Connected';
                statusBarItem.tooltip = 'Connected to PlatformFoundry';
                statusBarItem.backgroundColor = undefined;
                break;
            case 'disconnected':
                statusBarItem.text = '$(cloud-offline) PF: Disconnected';
                statusBarItem.tooltip = 'Click to connect';
                statusBarItem.backgroundColor = undefined;
                break;
            case 'error':
                statusBarItem.text = '$(error) PF: Error';
                statusBarItem.tooltip = 'Connection error';
                statusBarItem.backgroundColor = new vscode.ThemeColor('statusBarItem.errorBackground');
                break;
        }
    }

    function isPlatformFoundryFile(document: vscode.TextDocument): boolean {
        const fileName = document.fileName;
        return fileName.endsWith('platform.yaml') ||
               fileName.endsWith('score.yaml') ||
               fileName.endsWith('.pf.yaml');
    }

    function showPlanResult(result: any) {
        const panel = vscode.window.createWebviewPanel(
            'platformfoundryPlan',
            'PlatformFoundry Plan',
            vscode.ViewColumn.Beside,
            {}
        );

        panel.webview.html = `
            <!DOCTYPE html>
            <html>
            <head>
                <style>
                    body { font-family: var(--vscode-font-family); padding: 20px; }
                    .create { color: #4caf50; }
                    .update { color: #ff9800; }
                    .delete { color: #f44336; }
                    h2 { margin-top: 20px; }
                    ul { list-style-type: none; padding: 0; }
                    li { padding: 5px 0; }
                </style>
            </head>
            <body>
                <h1>Plan Result</h1>
                ${result.toCreate?.length ? `
                    <h2 class="create">To Create (${result.toCreate.length})</h2>
                    <ul>${result.toCreate.map((r: string) => `<li>+ ${r}</li>`).join('')}</ul>
                ` : ''}
                ${result.toUpdate?.length ? `
                    <h2 class="update">To Update (${result.toUpdate.length})</h2>
                    <ul>${result.toUpdate.map((r: string) => `<li>~ ${r}</li>`).join('')}</ul>
                ` : ''}
                ${result.toDelete?.length ? `
                    <h2 class="delete">To Delete (${result.toDelete.length})</h2>
                    <ul>${result.toDelete.map((r: string) => `<li>- ${r}</li>`).join('')}</ul>
                ` : ''}
                ${result.noChanges ? '<p>No changes detected.</p>' : ''}
            </body>
            </html>
        `;
    }
}

export function deactivate() {
    if (client) {
        client.disconnect();
    }
}
