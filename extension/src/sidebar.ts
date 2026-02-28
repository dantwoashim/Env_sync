import * as vscode from 'vscode';
import { execSync } from 'child_process';

export function registerSidebar(context: vscode.ExtensionContext) {
    const peersProvider = new EnvSyncTreeProvider('peers');
    const auditProvider = new EnvSyncTreeProvider('audit');

    vscode.window.registerTreeDataProvider('envsync.peers', peersProvider);
    vscode.window.registerTreeDataProvider('envsync.audit', auditProvider);

    // Refresh on interval
    setInterval(() => {
        peersProvider.refresh();
        auditProvider.refresh();
    }, 30000);
}

class EnvSyncTreeProvider implements vscode.TreeDataProvider<EnvSyncItem> {
    private _onDidChangeTreeData = new vscode.EventEmitter<void>();
    readonly onDidChangeTreeData = this._onDidChangeTreeData.event;
    private type: 'peers' | 'audit';

    constructor(type: 'peers' | 'audit') {
        this.type = type;
    }

    refresh() {
        this._onDidChangeTreeData.fire();
    }

    getTreeItem(element: EnvSyncItem): vscode.TreeItem {
        return element;
    }

    async getChildren(): Promise<EnvSyncItem[]> {
        try {
            if (this.type === 'peers') {
                return this.getPeers();
            }
            return this.getAuditEntries();
        } catch {
            return [new EnvSyncItem('EnvSync CLI not found', 'Install from envsync.dev')];
        }
    }

    private getPeers(): EnvSyncItem[] {
        try {
            const output = execSync('envsync peers --no-color', {
                timeout: 5000,
                encoding: 'utf-8',
            });

            const lines = output.split('\n').filter(l => l.trim());
            return lines.map(l => new EnvSyncItem(l.trim(), ''));
        } catch {
            return [new EnvSyncItem('No peers', 'Run envsync invite @teammate')];
        }
    }

    private getAuditEntries(): EnvSyncItem[] {
        try {
            const output = execSync('envsync audit --last 5 --no-color', {
                timeout: 5000,
                encoding: 'utf-8',
            });

            const lines = output.split('\n').filter(l => l.trim());
            return lines.map(l => new EnvSyncItem(l.trim(), ''));
        } catch {
            return [new EnvSyncItem('No events', 'Push or pull to create audit entries')];
        }
    }
}

class EnvSyncItem extends vscode.TreeItem {
    constructor(label: string, description: string) {
        super(label, vscode.TreeItemCollapsibleState.None);
        this.description = description;
    }
}
