import { Hono } from 'hono';
import type { Env } from '../types';

export const billingRoutes = new Hono<{ Bindings: Env }>();

// POST /checkout — Create a Stripe Checkout session
billingRoutes.post('/checkout', async (c) => {
    const body = await c.req.json<{
        team_id: string;
        plan: 'team' | 'enterprise';
        success_url?: string;
        cancel_url?: string;
    }>();

    if (!body.team_id || !body.plan) {
        return c.json({ error: 'missing_fields', message: 'team_id and plan required' }, 400);
    }

    // Stripe price IDs (would be env vars in production)
    const priceMap: Record<string, string> = {
        team: 'price_team_monthly_2900',
        enterprise: 'price_enterprise_monthly_19900',
    };

    const priceId = priceMap[body.plan];
    if (!priceId) {
        return c.json({ error: 'invalid_plan', message: 'Plan must be "team" or "enterprise"' }, 400);
    }

    // In production, this would call Stripe API
    // For now, return a mock checkout URL
    const checkoutUrl = `https://checkout.stripe.com/c/pay/cs_test_${body.team_id}_${body.plan}`;

    return c.json({
        checkout_url: checkoutUrl,
        plan: body.plan,
        team_id: body.team_id,
    });
});

// POST /webhook — Stripe webhook handler
billingRoutes.post('/webhook', async (c) => {
    const body = await c.req.text();
    const signature = c.req.header('Stripe-Signature') || '';

    // In production: verify Stripe webhook signature
    // const event = stripe.webhooks.constructEvent(body, signature, webhookSecret);

    try {
        const event = JSON.parse(body);

        switch (event.type) {
            case 'checkout.session.completed': {
                const session = event.data.object;
                const teamId = session.metadata?.team_id;
                const plan = session.metadata?.plan || 'team';

                if (teamId) {
                    // Upgrade team tier
                    await c.env.ENVSYNC_DATA.put(`team:${teamId}:tier`, plan);
                    await c.env.ENVSYNC_DATA.put(
                        `team:${teamId}:stripe_sub`,
                        session.subscription || ''
                    );
                    await c.env.ENVSYNC_DATA.put(
                        `team:${teamId}:tier_updated_at`,
                        String(Math.floor(Date.now() / 1000))
                    );
                }
                break;
            }

            case 'customer.subscription.updated': {
                const sub = event.data.object;
                const teamId = sub.metadata?.team_id;
                if (teamId && sub.status === 'active') {
                    const plan = sub.metadata?.plan || 'team';
                    await c.env.ENVSYNC_DATA.put(`team:${teamId}:tier`, plan);
                }
                break;
            }

            case 'customer.subscription.deleted': {
                const sub = event.data.object;
                const teamId = sub.metadata?.team_id;
                if (teamId) {
                    // Downgrade to free
                    await c.env.ENVSYNC_DATA.put(`team:${teamId}:tier`, 'free');
                    await c.env.ENVSYNC_DATA.delete(`team:${teamId}:stripe_sub`);
                }
                break;
            }
        }

        return c.json({ received: true });
    } catch {
        return c.json({ error: 'invalid_payload' }, 400);
    }
});

// GET /status/:team — Current tier and usage
billingRoutes.get('/status/:team', async (c) => {
    const teamId = c.req.param('team');

    const tier = await c.env.ENVSYNC_DATA.get(`team:${teamId}:tier`) || 'free';
    const stripeSub = await c.env.ENVSYNC_DATA.get(`team:${teamId}:stripe_sub`) || '';
    const updatedAt = await c.env.ENVSYNC_DATA.get(`team:${teamId}:tier_updated_at`) || '';

    // Get usage stats
    const dateKey = new Date().toISOString().split('T')[0];
    const blobCount = parseInt(
        await c.env.ENVSYNC_DATA.get(`ratelimit:blob:${teamId}:${dateKey}`) || '0'
    );

    // Team member count
    const teamData = await c.env.ENVSYNC_DATA.get(`team:${teamId}`);
    let memberCount = 0;
    if (teamData) {
        const team = JSON.parse(teamData);
        memberCount = team.members?.length || 0;
    }

    // Tier limits
    const limits: Record<string, { members: number; blobs_per_day: number; history_days: number }> = {
        free: { members: 3, blobs_per_day: 10, history_days: 3 },
        team: { members: -1, blobs_per_day: -1, history_days: 30 },
        enterprise: { members: -1, blobs_per_day: -1, history_days: 365 },
    };

    return c.json({
        team_id: teamId,
        tier,
        stripe_subscription: stripeSub,
        updated_at: updatedAt ? parseInt(updatedAt) : null,
        usage: {
            members: memberCount,
            blobs_today: blobCount,
        },
        limits: limits[tier] || limits.free,
    });
});
