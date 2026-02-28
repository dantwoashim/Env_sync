import type { Context, Next } from 'hono';

/**
 * CORS middleware configuration for future web dashboard.
 * 
 * Currently allows all origins for CLI usage.
 * When a web dashboard is added, restrict to specific origins.
 */
export function corsConfig() {
    return {
        origin: '*',
        allowMethods: ['GET', 'POST', 'PUT', 'DELETE', 'OPTIONS'],
        allowHeaders: ['Authorization', 'Content-Type', 'X-Envsync-Signature', 'X-Envsync-Timestamp'],
        exposeHeaders: ['X-RateLimit-Limit', 'X-RateLimit-Remaining', 'X-Request-ID'],
        maxAge: 86400, // Preflight cache: 24 hours
    };
}

/**
 * Request ID middleware — adds a unique request ID for tracing.
 */
export async function requestIdMiddleware(c: Context, next: Next) {
    const requestId = crypto.randomUUID();
    c.header('X-Request-ID', requestId);
    await next();
}
