import * as vscode from 'vscode';
import { registerCommands } from './commands';
import { createStatusBar } from './statusbar';
import { registerSidebar } from './sidebar';

let statusBarItem: vscode.StatusBarItem;

export function activate(context: vscode.ExtensionContext) {
    console.log('EnvSync extension activated');

    // Register commands
    registerCommands(context);

    // Create status bar
    statusBarItem = createStatusBar(context);

    // Register sidebar views
    registerSidebar(context);

    // Check sync status on activation
    checkSyncStatus();
}

async function checkSyncStatus() {
    try {
        const { execSync } = require('child_process');
        execSync('envsync version --short', { timeout: 5000 });
        statusBarItem.text = '$(check) EnvSync';
        statusBarItem.tooltip = 'EnvSync: Synced';
        statusBarItem.color = '#10B981';
    } catch {
        statusBarItem.text = '$(warning) EnvSync';
        statusBarItem.tooltip = 'EnvSync: CLI not found. Install from envsync.dev';
        statusBarItem.color = '#F59E0B';
    }
}

export function deactivate() {
    if (statusBarItem) {
        statusBarItem.dispose();
    }
}
