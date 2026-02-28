import { Hono } from 'hono';
import type { Env, Team, TeamMember } from '../types';

export const teamRoutes = new Hono<{ Bindings: Env }>();

// GET /:team/members — List team members
teamRoutes.get('/:team/members', async (c) => {
    const teamId = c.req.param('team');

    const data = await c.env.ENVSYNC_DATA.get(`team:${teamId}`);
    if (!data) {
        return c.json({ members: [] });
    }

    const team: Team = JSON.parse(data);
    return c.json({ members: team.members });
});

// PUT /:team/members/:user — Add or update a member
teamRoutes.put('/:team/members/:user', async (c) => {
    const teamId = c.req.param('team');
    const username = c.req.param('user');

    const body = await c.req.json<{
        fingerprint: string;
        public_key: string;
        role?: 'owner' | 'member';
    }>();

    if (!body.fingerprint || !body.public_key) {
        return c.json({ error: 'missing_fields', message: 'fingerprint and public_key required' }, 400);
    }

    // Load or create team
    const data = await c.env.ENVSYNC_DATA.get(`team:${teamId}`);
    let team: Team;

    if (data) {
        team = JSON.parse(data);
    } else {
        team = {
            id: teamId,
            name: teamId,
            members: [],
            created_at: Math.floor(Date.now() / 1000),
        };
    }

    // Check member limit (free tier: 3)
    const existingIdx = team.members.findIndex(m => m.username === username);
    if (existingIdx < 0 && team.members.length >= 3) {
        return c.json({
            error: 'member_limit',
            message: 'Free tier: maximum 3 team members. Upgrade at https://envsync.dev/pricing',
        }, 429);
    }

    const member: TeamMember = {
        username,
        fingerprint: body.fingerprint,
        public_key: body.public_key,
        role: body.role || 'member',
        added_at: Math.floor(Date.now() / 1000),
    };

    if (existingIdx >= 0) {
        team.members[existingIdx] = member;
    } else {
        team.members.push(member);
    }

    await c.env.ENVSYNC_DATA.put(`team:${teamId}`, JSON.stringify(team));

    return c.json({ status: 'added', member_count: team.members.length });
});

// DELETE /:team/members/:user — Remove a member
teamRoutes.delete('/:team/members/:user', async (c) => {
    const teamId = c.req.param('team');
    const username = c.req.param('user');

    const data = await c.env.ENVSYNC_DATA.get(`team:${teamId}`);
    if (!data) {
        return c.json({ error: 'not_found', message: 'Team not found' }, 404);
    }

    const team: Team = JSON.parse(data);
    const before = team.members.length;
    team.members = team.members.filter(m => m.username !== username);

    if (team.members.length === before) {
        return c.json({ error: 'not_found', message: `User @${username} not in team` }, 404);
    }

    await c.env.ENVSYNC_DATA.put(`team:${teamId}`, JSON.stringify(team));

    return c.json({ status: 'removed', member_count: team.members.length });
});
