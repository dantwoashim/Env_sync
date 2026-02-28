import { Hono } from 'hono';
import { cors } from 'hono/cors';
import type { Env } from './types';
import { healthRoutes } from './routes/health';
import { inviteRoutes } from './routes/invites';
import { relayRoutes } from './routes/relay';
import { teamRoutes } from './routes/teams';
import { billingRoutes } from './routes/billing';

const app = new Hono<{ Bindings: Env }>();

// Global middleware
app.use('*', cors({
    origin: '*',
    allowMethods: ['GET', 'POST', 'PUT', 'DELETE'],
    allowHeaders: ['Authorization', 'Content-Type'],
}));

// Global error handler
app.onError((err, c) => {
    console.error('Unhandled error:', err);
    return c.json({
        error: 'internal_error',
        message: 'An unexpected error occurred',
    }, 500);
});

// Mount routes
app.route('/health', healthRoutes);
app.route('/invites', inviteRoutes);
app.route('/relay', relayRoutes);
app.route('/teams', teamRoutes);
app.route('/billing', billingRoutes);

// 404 handler
app.notFound((c) => {
    return c.json({
        error: 'not_found',
        message: `Route ${c.req.method} ${c.req.path} not found`,
    }, 404);
});

// Export for Cloudflare Workers
export default app;

// Export Durable Object
export { SignalingRoom } from './durable/signaling-room';
