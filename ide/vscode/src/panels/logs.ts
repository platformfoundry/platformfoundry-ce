import * as vscode from 'vscode';
import { PlatformFoundryClient } from '../client';

export class LogsPanel {
    public static currentPanel: LogsPanel | undefined;
    private static readonly viewType = 'platformfoundryLogs';

    private readonly _panel: vscode.WebviewPanel;
    private readonly _extensionUri: vscode.Uri;
    private _disposables: vscode.Disposable[] = [];
    private _refreshInterval: NodeJS.Timeout | undefined;

    public static createOrShow(extensionUri: vscode.Uri, client: PlatformFoundryClient, workloadName: string) {
        const column = vscode.window.activeTextEditor
            ? vscode.window.activeTextEditor.viewColumn
            : undefined;

        if (LogsPanel.currentPanel) {
            LogsPanel.currentPanel._panel.reveal(column);
            LogsPanel.currentPanel.loadLogs(client, workloadName);
            return;
        }

        const panel = vscode.window.createWebviewPanel(
            LogsPanel.viewType,
            `Logs: ${workloadName}`,
            column || vscode.ViewColumn.One,
            {
                enableScripts: true,
                retainContextWhenHidden: true
            }
        );

        LogsPanel.currentPanel = new LogsPanel(panel, extensionUri, client, workloadName);
    }

    private constructor(
        panel: vscode.WebviewPanel,
        extensionUri: vscode.Uri,
        client: PlatformFoundryClient,
        workloadName: string
    ) {
        this._panel = panel;
        this._extensionUri = extensionUri;

        this._panel.webview.html = this._getHtmlForWebview(workloadName);

        this._panel.onDidDispose(() => this.dispose(), null, this._disposables);

        this._panel.webview.onDidReceiveMessage(
            async message => {
                switch (message.command) {
                    case 'refresh':
                        await this.loadLogs(client, workloadName);
                        break;
                    case 'toggleFollow':
                        if (message.follow) {
                            this.startFollowing(client, workloadName);
                        } else {
                            this.stopFollowing();
                        }
                        break;
                }
            },
            null,
            this._disposables
        );

        this.loadLogs(client, workloadName);
    }

    private async loadLogs(client: PlatformFoundryClient, workloadName: string) {
        try {
            const logs = await client.getLogs(workloadName, { tail: 100 });
            this._panel.webview.postMessage({ command: 'logs', data: logs });
        } catch (error) {
            this._panel.webview.postMessage({ command: 'error', data: `Failed to load logs: ${error}` });
        }
    }

    private startFollowing(client: PlatformFoundryClient, workloadName: string) {
        this.stopFollowing();
        this._refreshInterval = setInterval(() => {
            this.loadLogs(client, workloadName);
        }, 2000);
    }

    private stopFollowing() {
        if (this._refreshInterval) {
            clearInterval(this._refreshInterval);
            this._refreshInterval = undefined;
        }
    }

    public dispose() {
        LogsPanel.currentPanel = undefined;
        this.stopFollowing();
        this._panel.dispose();

        while (this._disposables.length) {
            const disposable = this._disposables.pop();
            if (disposable) {
                disposable.dispose();
            }
        }
    }

    private _getHtmlForWebview(workloadName: string): string {
        return `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Logs: ${workloadName}</title>
    <style>
        body {
            font-family: var(--vscode-editor-font-family);
            font-size: var(--vscode-editor-font-size);
            padding: 0;
            margin: 0;
            background: var(--vscode-editor-background);
            color: var(--vscode-editor-foreground);
        }
        .toolbar {
            position: sticky;
            top: 0;
            background: var(--vscode-editor-background);
            padding: 10px;
            border-bottom: 1px solid var(--vscode-panel-border);
            display: flex;
            gap: 10px;
            align-items: center;
        }
        .toolbar button {
            background: var(--vscode-button-background);
            color: var(--vscode-button-foreground);
            border: none;
            padding: 6px 12px;
            cursor: pointer;
            border-radius: 3px;
        }
        .toolbar button:hover {
            background: var(--vscode-button-hoverBackground);
        }
        .toolbar label {
            display: flex;
            align-items: center;
            gap: 5px;
        }
        #logs {
            padding: 10px;
            white-space: pre-wrap;
            word-wrap: break-word;
            font-family: monospace;
        }
        .log-line {
            padding: 2px 0;
        }
        .log-timestamp {
            color: var(--vscode-descriptionForeground);
        }
        .error {
            color: var(--vscode-errorForeground);
            padding: 10px;
        }
    </style>
</head>
<body>
    <div class="toolbar">
        <span><strong>Logs:</strong> ${workloadName}</span>
        <button id="refreshBtn">Refresh</button>
        <label>
            <input type="checkbox" id="followCheckbox">
            Follow
        </label>
    </div>
    <div id="logs">Loading...</div>

    <script>
        const vscode = acquireVsCodeApi();
        const logsDiv = document.getElementById('logs');
        const refreshBtn = document.getElementById('refreshBtn');
        const followCheckbox = document.getElementById('followCheckbox');

        refreshBtn.addEventListener('click', () => {
            vscode.postMessage({ command: 'refresh' });
        });

        followCheckbox.addEventListener('change', () => {
            vscode.postMessage({ command: 'toggleFollow', follow: followCheckbox.checked });
        });

        window.addEventListener('message', event => {
            const message = event.data;
            switch (message.command) {
                case 'logs':
                    logsDiv.innerHTML = formatLogs(message.data);
                    if (followCheckbox.checked) {
                        logsDiv.scrollTop = logsDiv.scrollHeight;
                    }
                    break;
                case 'error':
                    logsDiv.innerHTML = '<div class="error">' + message.data + '</div>';
                    break;
            }
        });

        function formatLogs(logs) {
            if (!logs) return '<div class="log-line">No logs available</div>';
            return logs.split('\\n').map(line => {
                const match = line.match(/^\\[(.*?)\\]/);
                if (match) {
                    const timestamp = match[1];
                    const rest = line.substring(match[0].length);
                    return '<div class="log-line"><span class="log-timestamp">[' + timestamp + ']</span>' + rest + '</div>';
                }
                return '<div class="log-line">' + line + '</div>';
            }).join('');
        }
    </script>
</body>
</html>`;
    }
}
