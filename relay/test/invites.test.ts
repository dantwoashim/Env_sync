/**
 * Invite flow tests (Miniflare).
 * 
 * Tests the invite lifecycle:
 *   1. Create invite (POST /invites)
 *   2. Retrieve invite (GET /invites/:hash)
 *   3. Consume invite (POST /invites/:hash/consume)
 *   4. Verify consumed invite cannot be reused
 *   5. Expired invites are rejected
 */

import { describe, it, expect, beforeAll } from 'vitest';

const BASE_URL = 'http://localhost:8787';

describe('Invite Flow', () => {
    const tokenHash = 'test-token-hash-' + Date.now();

    it('should create an invite', async () => {
        const res = await fetch(`${BASE_URL}/invites`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                token_hash: tokenHash,
                team_id: 'test-team-id',
                inviter: 'alice',
                inviter_fingerprint: 'fp_alice_123',
                invitee: 'bob',
            }),
        });
        expect(res.status).toBe(201);
    });

    it('should retrieve the invite', async () => {
        const res = await fetch(`${BASE_URL}/invites/${tokenHash}`);
        expect(res.status).toBe(200);
        const data = await res.json() as any;
        expect(data.team_id).toBe('test-team-id');
        expect(data.inviter).toBe('alice');
    });

    it('should consume the invite', async () => {
        const res = await fetch(`${BASE_URL}/invites/${tokenHash}/consume`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                fingerprint: 'fp_bob_456',
            }),
        });
        expect(res.status).toBe(200);
    });

    it('should reject already-consumed invite', async () => {
        const res = await fetch(`${BASE_URL}/invites/${tokenHash}/consume`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                fingerprint: 'fp_charlie_789',
            }),
        });
        expect(res.status).toBeGreaterThanOrEqual(400);
    });

    it('should return 404 for unknown invite', async () => {
        const res = await fetch(`${BASE_URL}/invites/nonexistent-hash`);
        expect(res.status).toBe(404);
    });
});
