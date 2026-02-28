import * as vscode from 'vscode';
import { execSync } from 'child_process';

export function registerCommands(context: vscode.ExtensionContext) {
    context.subscriptions.push(
        vscode.commands.registerCommand('envsync.push', runPush),
        vscode.commands.registerCommand('envsync.pull', runPull),
        vscode.commands.registerCommand('envsync.diff', runDiff),
        vscode.commands.registerCommand('envsync.audit', runAudit),
        vscode.commands.registerCommand('envsync.peers', runPeers),
    );
}

async function runPush() {
    const terminal = vscode.window.createTerminal('EnvSync Push');
    terminal.show();
    terminal.sendText('envsync push');
    vscode.window.showInformationMessage('EnvSync: Pushing .env to peers...');
}

async function runPull() {
    const terminal = vscode.window.createTerminal('EnvSync Pull');
    terminal.show();
    terminal.sendText('envsync pull --timeout 10');
    vscode.window.showInformationMessage('EnvSync: Pulling .env from peers...');
}

async function runDiff() {
    try {
        const cwd = vscode.workspace.workspaceFolders?.[0]?.uri.fsPath;
        if (!cwd) {
            vscode.window.showWarningMessage('No workspace folder open');
            return;
        }

        const output = execSync('envsync diff --no-color', {
            cwd,
            timeout: 10000,
            encoding: 'utf-8',
        });

        const doc = await vscode.workspace.openTextDocument({
            content: output,
            language: 'diff',
        });
        await vscode.window.showTextDocument(doc);
    } catch (err: any) {
        vscode.window.showErrorMessage(`EnvSync Diff failed: ${err.message}`);
    }
}

async function runAudit() {
    try {
        const output = execSync('envsync audit --last 20 --no-color', {
            timeout: 10000,
            encoding: 'utf-8',
        });

        const doc = await vscode.workspace.openTextDocument({
            content: output,
            language: 'plaintext',
        });
        await vscode.window.showTextDocument(doc);
    } catch (err: any) {
        vscode.window.showErrorMessage(`EnvSync Audit failed: ${err.message}`);
    }
}

async function runPeers() {
    try {
        const output = execSync('envsync peers --no-color', {
            timeout: 10000,
            encoding: 'utf-8',
        });

        const doc = await vscode.workspace.openTextDocument({
            content: output,
            language: 'plaintext',
        });
        await vscode.window.showTextDocument(doc);
    } catch (err: any) {
        vscode.window.showErrorMessage(`EnvSync Peers failed: ${err.message}`);
    }
}
