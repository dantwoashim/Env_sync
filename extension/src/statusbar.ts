import * as vscode from 'vscode';

export function createStatusBar(context: vscode.ExtensionContext): vscode.StatusBarItem {
    const item = vscode.window.createStatusBarItem(
        vscode.StatusBarAlignment.Right,
        100,
    );

    item.text = '$(sync) EnvSync';
    item.tooltip = 'EnvSync: Click for options';
    item.command = 'envsync.showQuickPick';

    // Register the quick pick command
    context.subscriptions.push(
        vscode.commands.registerCommand('envsync.showQuickPick', showQuickPick),
    );

    item.show();
    context.subscriptions.push(item);

    return item;
}

async function showQuickPick() {
    const items: vscode.QuickPickItem[] = [
        { label: '$(cloud-upload) Push', description: 'Push .env to peers', detail: 'envsync push' },
        { label: '$(cloud-download) Pull', description: 'Pull .env from peers', detail: 'envsync pull' },
        { label: '$(diff) Diff', description: 'Compare local vs synced', detail: 'envsync diff' },
        { label: '$(person) Peers', description: 'List team members', detail: 'envsync peers' },
        { label: '$(history) Audit', description: 'View sync history', detail: 'envsync audit' },
    ];

    const selected = await vscode.window.showQuickPick(items, {
        placeHolder: 'EnvSync: Choose an action',
    });

    if (selected) {
        switch (selected.detail) {
            case 'envsync push':
                vscode.commands.executeCommand('envsync.push');
                break;
            case 'envsync pull':
                vscode.commands.executeCommand('envsync.pull');
                break;
            case 'envsync diff':
                vscode.commands.executeCommand('envsync.diff');
                break;
            case 'envsync peers':
                vscode.commands.executeCommand('envsync.peers');
                break;
            case 'envsync audit':
                vscode.commands.executeCommand('envsync.audit');
                break;
        }
    }
}
