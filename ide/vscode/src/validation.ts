import * as vscode from 'vscode';
import * as yaml from 'yaml';
import { PlatformFoundryClient } from './client';

const diagnosticCollection = vscode.languages.createDiagnosticCollection('platformfoundry');

export async function validateDocument(
    document: vscode.TextDocument,
    client: PlatformFoundryClient
): Promise<vscode.Diagnostic[]> {
    const diagnostics: vscode.Diagnostic[] = [];
    const text = document.getText();

    // Parse YAML
    let parsed: any;
    try {
        parsed = yaml.parse(text);
    } catch (e: any) {
        const diagnostic = new vscode.Diagnostic(
            new vscode.Range(0, 0, 0, 0),
            `YAML parse error: ${e.message}`,
            vscode.DiagnosticSeverity.Error
        );
        diagnostics.push(diagnostic);
        diagnosticCollection.set(document.uri, diagnostics);
        return diagnostics;
    }

    if (!parsed) {
        diagnosticCollection.set(document.uri, []);
        return [];
    }

    // Validate structure
    diagnostics.push(...validateStructure(document, parsed));

    // Validate with server if connected
    if (client.isConnected()) {
        try {
            const result = await client.validate(text);
            if (!result.valid && result.errors) {
                for (const error of result.errors) {
                    const range = findRange(document, error.path);
                    const diagnostic = new vscode.Diagnostic(
                        range,
                        error.message,
                        severityFromString(error.severity)
                    );
                    diagnostics.push(diagnostic);
                }
            }
        } catch (e) {
            // Server validation failed, continue with local validation only
        }
    }

    diagnosticCollection.set(document.uri, diagnostics);
    return diagnostics;
}

function validateStructure(document: vscode.TextDocument, parsed: any): vscode.Diagnostic[] {
    const diagnostics: vscode.Diagnostic[] = [];

    // Check apiVersion
    if (!parsed.apiVersion) {
        diagnostics.push(createDiagnostic(
            document,
            'apiVersion',
            'Missing required field: apiVersion',
            vscode.DiagnosticSeverity.Error
        ));
    } else if (!parsed.apiVersion.startsWith('platformfoundry.io/')) {
        const range = findFieldRange(document, 'apiVersion');
        diagnostics.push(new vscode.Diagnostic(
            range,
            'apiVersion should start with "platformfoundry.io/"',
            vscode.DiagnosticSeverity.Warning
        ));
    }

    // Check kind
    if (!parsed.kind) {
        diagnostics.push(createDiagnostic(
            document,
            'kind',
            'Missing required field: kind',
            vscode.DiagnosticSeverity.Error
        ));
    } else {
        const validKinds = [
            'Platform', 'Workload', 'Environment', 'Service',
            'Deployment', 'Resource', 'Policy', 'Workflow',
            'Secret', 'ConfigMap', 'Gateway', 'Route'
        ];
        if (!validKinds.includes(parsed.kind)) {
            const range = findFieldRange(document, 'kind');
            diagnostics.push(new vscode.Diagnostic(
                range,
                `Unknown kind: ${parsed.kind}. Valid kinds: ${validKinds.join(', ')}`,
                vscode.DiagnosticSeverity.Warning
            ));
        }
    }

    // Check metadata
    if (!parsed.metadata) {
        diagnostics.push(createDiagnostic(
            document,
            'metadata',
            'Missing required field: metadata',
            vscode.DiagnosticSeverity.Error
        ));
    } else if (!parsed.metadata.name) {
        diagnostics.push(createDiagnostic(
            document,
            'metadata.name',
            'Missing required field: metadata.name',
            vscode.DiagnosticSeverity.Error
        ));
    } else {
        // Validate name format
        const namePattern = /^[a-z0-9][a-z0-9-]*[a-z0-9]$|^[a-z0-9]$/;
        if (!namePattern.test(parsed.metadata.name)) {
            const range = findFieldRange(document, 'name', 'metadata');
            diagnostics.push(new vscode.Diagnostic(
                range,
                'Name must be lowercase alphanumeric with hyphens, starting and ending with alphanumeric',
                vscode.DiagnosticSeverity.Error
            ));
        }
    }

    // Check spec
    if (!parsed.spec) {
        diagnostics.push(createDiagnostic(
            document,
            'spec',
            'Missing required field: spec',
            vscode.DiagnosticSeverity.Warning
        ));
    }

    // Kind-specific validation
    if (parsed.kind && parsed.spec) {
        diagnostics.push(...validateKindSpec(document, parsed.kind, parsed.spec));
    }

    return diagnostics;
}

function validateKindSpec(document: vscode.TextDocument, kind: string, spec: any): vscode.Diagnostic[] {
    const diagnostics: vscode.Diagnostic[] = [];

    switch (kind) {
        case 'Workload':
            if (!spec.container && !spec.containers) {
                diagnostics.push(createDiagnostic(
                    document,
                    'spec.container',
                    'Workload requires container or containers specification',
                    vscode.DiagnosticSeverity.Error
                ));
            }
            if (spec.replicas !== undefined && (spec.replicas < 0 || !Number.isInteger(spec.replicas))) {
                const range = findFieldRange(document, 'replicas', 'spec');
                diagnostics.push(new vscode.Diagnostic(
                    range,
                    'replicas must be a non-negative integer',
                    vscode.DiagnosticSeverity.Error
                ));
            }
            break;

        case 'Environment':
            if (!spec.cluster) {
                diagnostics.push(createDiagnostic(
                    document,
                    'spec.cluster',
                    'Environment requires cluster specification',
                    vscode.DiagnosticSeverity.Error
                ));
            }
            break;

        case 'Policy':
            if (!spec.rules && !spec.rego) {
                diagnostics.push(createDiagnostic(
                    document,
                    'spec.rules',
                    'Policy requires rules or rego specification',
                    vscode.DiagnosticSeverity.Error
                ));
            }
            break;

        case 'Workflow':
            if (!spec.steps || !Array.isArray(spec.steps) || spec.steps.length === 0) {
                diagnostics.push(createDiagnostic(
                    document,
                    'spec.steps',
                    'Workflow requires at least one step',
                    vscode.DiagnosticSeverity.Error
                ));
            }
            break;
    }

    return diagnostics;
}

function createDiagnostic(
    document: vscode.TextDocument,
    field: string,
    message: string,
    severity: vscode.DiagnosticSeverity
): vscode.Diagnostic {
    const range = findFieldRange(document, field);
    return new vscode.Diagnostic(range, message, severity);
}

function findFieldRange(document: vscode.TextDocument, field: string, parent?: string): vscode.Range {
    const text = document.getText();
    const lines = text.split('\n');

    let inParent = !parent;
    let parentIndent = 0;

    for (let i = 0; i < lines.length; i++) {
        const line = lines[i];
        const trimmed = line.trimStart();
        const indent = line.length - trimmed.length;

        if (parent && trimmed.startsWith(parent + ':')) {
            inParent = true;
            parentIndent = indent;
            continue;
        }

        if (inParent && parent && indent <= parentIndent && trimmed.length > 0 && !trimmed.startsWith(parent + ':')) {
            inParent = false;
        }

        if (inParent && trimmed.startsWith(field + ':')) {
            const start = line.indexOf(field);
            return new vscode.Range(i, start, i, line.length);
        }
    }

    return new vscode.Range(0, 0, 0, 0);
}

function findRange(document: vscode.TextDocument, path?: string): vscode.Range {
    if (!path) {
        return new vscode.Range(0, 0, 0, 0);
    }

    const parts = path.split('.');
    if (parts.length === 1) {
        return findFieldRange(document, parts[0]);
    }

    return findFieldRange(document, parts[parts.length - 1], parts[parts.length - 2]);
}

function severityFromString(severity?: string): vscode.DiagnosticSeverity {
    switch (severity) {
        case 'error':
            return vscode.DiagnosticSeverity.Error;
        case 'warning':
            return vscode.DiagnosticSeverity.Warning;
        case 'info':
            return vscode.DiagnosticSeverity.Information;
        default:
            return vscode.DiagnosticSeverity.Error;
    }
}

export function disposeDiagnostics(): void {
    diagnosticCollection.dispose();
}
