import { Hono } from 'hono';
import type { Env, BlobMetadata } from '../types';

export const relayRoutes = new Hono<{ Bindings: Env }>();

// PUT /:team/:blob — Store encrypted blob
relayRoutes.put('/:team/:blob', async (c) => {
    const teamId = c.req.param('team');
    const blobId = c.req.param('blob');
    const maxSize = parseInt(c.env.MAX_BLOB_SIZE || '65536');

    // Read body
    const body = await c.req.arrayBuffer();
    if (body.byteLength > maxSize) {
        return c.json({
            error: 'too_large',
            message: `Blob exceeds maximum size of ${maxSize} bytes`,
        }, 413);
    }

    // Rate limiting: max 10 blobs per team per day (free tier)
    const rateLimitKey = `ratelimit:blob:${teamId}:${dateKey()}`;
    const currentCount = parseInt(await c.env.ENVSYNC_DATA.get(rateLimitKey) || '0');
    if (currentCount >= 10) {
        return c.json({
            error: 'rate_limited',
            message: 'Free tier: maximum 10 relay blobs per day. Upgrade at https://envsync.dev/pricing',
        }, 429);
    }

    // Parse metadata from headers
    const metadata: BlobMetadata = {
        blob_id: blobId,
        team_id: teamId,
        sender_fingerprint: c.req.header('X-EnvSync-Sender') || '',
        recipient_fingerprint: c.req.header('X-EnvSync-Recipient') || '',
        ephemeral_public_key: c.req.header('X-EnvSync-EphemeralKey') || '',
        size: body.byteLength,
        uploaded_at: Math.floor(Date.now() / 1000),
        expires_at: Math.floor(Date.now() / 1000) + (parseInt(c.env.BLOB_TTL_HOURS || '72') * 3600),
        filename: c.req.header('X-EnvSync-Filename') || '.env',
    };

    if (!metadata.sender_fingerprint || !metadata.recipient_fingerprint) {
        return c.json({ error: 'missing_headers', message: 'X-EnvSync-Sender and X-EnvSync-Recipient headers required' }, 400);
    }

    const ttlHours = parseInt(c.env.BLOB_TTL_HOURS || '72');

    // Store blob data
    await c.env.ENVSYNC_DATA.put(
        `blob:${teamId}:${blobId}:data`,
        body,
        { expirationTtl: ttlHours * 3600 }
    );

    // Store metadata
    await c.env.ENVSYNC_DATA.put(
        `blob:${teamId}:${blobId}:meta`,
        JSON.stringify(metadata),
        { expirationTtl: ttlHours * 3600 }
    );

    // Add to pending list for recipient
    const pendingKey = `pending:${teamId}:${metadata.recipient_fingerprint}`;
    const pendingList = JSON.parse(await c.env.ENVSYNC_DATA.get(pendingKey) || '[]') as string[];
    if (!pendingList.includes(blobId)) {
        pendingList.push(blobId);
        await c.env.ENVSYNC_DATA.put(pendingKey, JSON.stringify(pendingList), { expirationTtl: ttlHours * 3600 });
    }

    // Increment rate limit counter
    await c.env.ENVSYNC_DATA.put(rateLimitKey, String(currentCount + 1), { expirationTtl: 86400 });

    return c.json({ status: 'stored', blob_id: blobId, expires_at: metadata.expires_at }, 201);
});

// GET /:team/pending — List pending blobs for a recipient
relayRoutes.get('/:team/pending', async (c) => {
    const teamId = c.req.param('team');
    const recipientFP = c.req.query('for');

    if (!recipientFP) {
        return c.json({ error: 'missing_param', message: '?for=fingerprint query parameter required' }, 400);
    }

    const pendingKey = `pending:${teamId}:${recipientFP}`;
    const pendingList = JSON.parse(await c.env.ENVSYNC_DATA.get(pendingKey) || '[]') as string[];

    // Fetch metadata for each pending blob
    const blobs: BlobMetadata[] = [];
    for (const blobId of pendingList) {
        const metaData = await c.env.ENVSYNC_DATA.get(`blob:${teamId}:${blobId}:meta`);
        if (metaData) {
            blobs.push(JSON.parse(metaData));
        }
    }

    return c.json({ pending: blobs });
});

// GET /:team/:blob — Download blob
relayRoutes.get('/:team/:blob', async (c) => {
    const teamId = c.req.param('team');
    const blobId = c.req.param('blob');

    // Get metadata
    const metaData = await c.env.ENVSYNC_DATA.get(`blob:${teamId}:${blobId}:meta`);
    if (!metaData) {
        return c.json({ error: 'not_found', message: 'Blob not found or expired' }, 404);
    }

    const metadata: BlobMetadata = JSON.parse(metaData);

    // Get blob data
    const data = await c.env.ENVSYNC_DATA.get(`blob:${teamId}:${blobId}:data`, 'arrayBuffer');
    if (!data) {
        return c.json({ error: 'not_found', message: 'Blob data not found' }, 404);
    }

    return new Response(data, {
        headers: {
            'Content-Type': 'application/octet-stream',
            'X-EnvSync-Sender': metadata.sender_fingerprint,
            'X-EnvSync-EphemeralKey': metadata.ephemeral_public_key,
            'X-EnvSync-Filename': metadata.filename,
            'X-EnvSync-UploadedAt': String(metadata.uploaded_at),
        },
    });
});

// DELETE /:team/:blob — Delete blob after download
relayRoutes.delete('/:team/:blob', async (c) => {
    const teamId = c.req.param('team');
    const blobId = c.req.param('blob');

    // Remove blob data and metadata
    await c.env.ENVSYNC_DATA.delete(`blob:${teamId}:${blobId}:data`);
    await c.env.ENVSYNC_DATA.delete(`blob:${teamId}:${blobId}:meta`);

    // Remove from pending lists (best effort)
    const recipientFP = c.req.header('X-EnvSync-Fingerprint') || '';
    if (recipientFP) {
        const pendingKey = `pending:${teamId}:${recipientFP}`;
        const pendingList = JSON.parse(await c.env.ENVSYNC_DATA.get(pendingKey) || '[]') as string[];
        const updated = pendingList.filter(id => id !== blobId);
        if (updated.length > 0) {
            await c.env.ENVSYNC_DATA.put(pendingKey, JSON.stringify(updated));
        } else {
            await c.env.ENVSYNC_DATA.delete(pendingKey);
        }
    }

    return c.json({ status: 'deleted' });
});

function dateKey(): string {
    return new Date().toISOString().split('T')[0]; // YYYY-MM-DD
}
