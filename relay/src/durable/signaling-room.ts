import type { SignalMessage } from '../types';

/**
 * SignalingRoom — Durable Object for TCP hole-punch signaling.
 * 
 * Each team gets one persistent room. Peers connect via WebSocket,
 * exchange endpoint information, then disconnect. The DO hibernates
 * when no connections are active (cost savings).
 */
export class SignalingRoom {
    private state: DurableObjectState;
    private sessions: Map<string, WebSocket> = new Map();

    constructor(state: DurableObjectState) {
        this.state = state;
    }

    async fetch(request: Request): Promise<Response> {
        const url = new URL(request.url);

        if (url.pathname === '/ws') {
            return this.handleWebSocket(request);
        }

        if (url.pathname === '/info') {
            return new Response(JSON.stringify({
                connected_peers: this.sessions.size,
                peers: Array.from(this.sessions.keys()),
            }), {
                headers: { 'Content-Type': 'application/json' },
            });
        }

        return new Response('Not found', { status: 404 });
    }

    private handleWebSocket(request: Request): Response {
        const fingerprint = new URL(request.url).searchParams.get('fp') || 'unknown';

        const pair = new WebSocketPair();
        const [client, server] = Object.values(pair);

        // Accept the WebSocket
        server.accept();

        // Store the session
        this.sessions.set(fingerprint, server);

        // Notify existing peers about the new connection
        const joinMsg = JSON.stringify({
            type: 'peer_joined',
            fingerprint,
            peer_count: this.sessions.size,
        });
        this.broadcast(joinMsg, fingerprint);

        // Handle incoming messages
        server.addEventListener('message', (event) => {
            try {
                const msg: SignalMessage = JSON.parse(event.data as string);
                msg.sender_fingerprint = fingerprint;

                // Forward to all other peers in the room
                this.broadcast(JSON.stringify(msg), fingerprint);
            } catch {
                server.send(JSON.stringify({ error: 'invalid_message' }));
            }
        });

        // Handle disconnect
        server.addEventListener('close', () => {
            this.sessions.delete(fingerprint);
            const leaveMsg = JSON.stringify({
                type: 'peer_left',
                fingerprint,
                peer_count: this.sessions.size,
            });
            this.broadcast(leaveMsg, fingerprint);
        });

        server.addEventListener('error', () => {
            this.sessions.delete(fingerprint);
        });

        return new Response(null, {
            status: 101,
            webSocket: client,
        });
    }

    /**
     * Broadcast a message to all connected peers except the sender.
     */
    private broadcast(message: string, excludeFingerprint: string): void {
        for (const [fp, ws] of this.sessions) {
            if (fp !== excludeFingerprint) {
                try {
                    ws.send(message);
                } catch {
                    // Remove dead connections
                    this.sessions.delete(fp);
                }
            }
        }
    }
}
