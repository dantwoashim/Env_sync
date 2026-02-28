import { Hono } from 'hono';
import type { Env, Invite } from '../types';

export const inviteRoutes = new Hono<{ Bindings: Env }>();

// POST / — Create a new invite
inviteRoutes.post('/', async (c) => {
    const body = await c.req.json<{
        token_hash: string;
        team_id: string;
        inviter: string;
        inviter_fingerprint: string;
        invitee: string;
        expected_fingerprint: string;
    }>();

    // Validate required fields
    if (!body.token_hash || !body.team_id || !body.invitee || !body.expected_fingerprint) {
        return c.json({ error: 'missing_fields', message: 'token_hash, team_id, invitee, and expected_fingerprint are required' }, 400);
    }

    // Check for duplicate
    const existing = await c.env.ENVSYNC_DATA.get(`invite:${body.token_hash}`);
    if (existing) {
        return c.json({ error: 'duplicate', message: 'An invite with this token already exists' }, 409);
    }

    const ttlHours = parseInt(c.env.INVITE_TTL_HOURS || '24');
    const now = Math.floor(Date.now() / 1000);

    const invite: Invite = {
        token_hash: body.token_hash,
        team_id: body.team_id,
        inviter: body.inviter,
        inviter_fingerprint: body.inviter_fingerprint,
        invitee: body.invitee,
        expected_fingerprint: body.expected_fingerprint,
        created_at: now,
        expires_at: now + (ttlHours * 3600),
        consumed: false,
    };

    // Store with TTL
    await c.env.ENVSYNC_DATA.put(
        `invite:${body.token_hash}`,
        JSON.stringify(invite),
        { expirationTtl: ttlHours * 3600 }
    );

    return c.json({ status: 'created', expires_at: invite.expires_at }, 201);
});

// GET /:hash — Retrieve an invite
inviteRoutes.get('/:hash', async (c) => {
    const hash = c.req.param('hash');
    const data = await c.env.ENVSYNC_DATA.get(`invite:${hash}`);

    if (!data) {
        return c.json({ error: 'not_found', message: 'Invite not found or expired' }, 404);
    }

    const invite: Invite = JSON.parse(data);

    if (invite.consumed) {
        return c.json({ error: 'consumed', message: 'This invite has already been used' }, 410);
    }

    // Don't leak the full invite data — only what the joiner needs
    return c.json({
        team_id: invite.team_id,
        inviter: invite.inviter,
        inviter_fingerprint: invite.inviter_fingerprint,
        expected_fingerprint: invite.expected_fingerprint,
        expires_at: invite.expires_at,
    });
});

// DELETE /:hash — Consume (redeem) an invite
// POST /:hash/consume — Alternative consume endpoint
inviteRoutes.post('/:hash/consume', consumeInvite);
inviteRoutes.delete('/:hash', consumeInvite);

async function consumeInvite(c: any) {
    const hash = c.req.param('hash');

    // Read the fingerprint of the person trying to join
    const joinerFingerprint = c.req.header('X-EnvSync-Fingerprint') || '';

    const data = await c.env.ENVSYNC_DATA.get(`invite:${hash}`);
    if (!data) {
        return c.json({ error: 'not_found', message: 'Invite not found or expired' }, 404);
    }

    const invite: Invite & { remaining_attempts?: number } = JSON.parse(data);

    if (invite.consumed) {
        return c.json({ error: 'consumed', message: 'This invite has already been used' }, 410);
    }

    // Initialize attempt counter if not present (backwards compat)
    if (invite.remaining_attempts === undefined) {
        invite.remaining_attempts = 3;
    }

    // Verify the joiner's fingerprint matches the expected one
    if (joinerFingerprint && invite.expected_fingerprint) {
        if (joinerFingerprint !== invite.expected_fingerprint) {
            // Decrement attempts instead of burning the token
            invite.remaining_attempts--;

            if (invite.remaining_attempts <= 0) {
                // Too many failed attempts — burn the token
                invite.consumed = true;
                await c.env.ENVSYNC_DATA.put(
                    `invite:${hash}`,
                    JSON.stringify(invite),
                    { expirationTtl: 60 }
                );
                return c.json({
                    error: 'max_attempts_exceeded',
                    message: 'Too many failed join attempts. Ask the inviter to regenerate the code.',
                }, 403);
            }

            // Save decremented counter
            const ttlRemaining = Math.max(invite.expires_at - Math.floor(Date.now() / 1000), 60);
            await c.env.ENVSYNC_DATA.put(
                `invite:${hash}`,
                JSON.stringify(invite),
                { expirationTtl: ttlRemaining }
            );

            return c.json({
                error: 'fingerprint_mismatch',
                message: 'Your SSH key fingerprint does not match the expected invitee',
                remaining_attempts: invite.remaining_attempts,
            }, 403);
        }
    }

    // Mark as consumed — correct fingerprint match
    invite.consumed = true;
    await c.env.ENVSYNC_DATA.put(
        `invite:${hash}`,
        JSON.stringify(invite),
        { expirationTtl: 60 } // Delete after 1 minute
    );

    return c.json({
        status: 'consumed',
        team_id: invite.team_id,
        inviter: invite.inviter,
        inviter_fingerprint: invite.inviter_fingerprint,
    });
}
