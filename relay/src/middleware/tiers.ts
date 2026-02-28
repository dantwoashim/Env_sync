import type { Env } from '../types';

export type TierName = 'free' | 'team' | 'enterprise';

export interface TierLimits {
    maxMembers: number;      // -1 = unlimited
    maxBlobsPerDay: number;  // -1 = unlimited
    blobTtlHours: number;
    historyDays: number;
}

const TIER_CONFIG: Record<TierName, TierLimits> = {
    free: {
        maxMembers: 3,
        maxBlobsPerDay: 10,
        blobTtlHours: 72,
        historyDays: 3,
    },
    team: {
        maxMembers: -1,
        maxBlobsPerDay: -1,
        blobTtlHours: 720, // 30 days
        historyDays: 30,
    },
    enterprise: {
        maxMembers: -1,
        maxBlobsPerDay: -1,
        blobTtlHours: 8760, // 365 days
        historyDays: 365,
    },
};

/**
 * Get the tier for a team.
 */
export async function getTeamTier(env: Env, teamId: string): Promise<TierName> {
    const tier = await env.ENVSYNC_DATA.get(`team:${teamId}:tier`);
    if (tier && (tier === 'team' || tier === 'enterprise')) {
        return tier;
    }
    return 'free';
}

/**
 * Get the limits for a team's tier.
 */
export async function getTeamLimits(env: Env, teamId: string): Promise<TierLimits> {
    const tier = await getTeamTier(env, teamId);
    return TIER_CONFIG[tier];
}

/**
 * Check if a team can add more members.
 */
export async function canAddMember(env: Env, teamId: string, currentCount: number): Promise<boolean> {
    const limits = await getTeamLimits(env, teamId);
    return limits.maxMembers < 0 || currentCount < limits.maxMembers;
}

/**
 * Check if a team can upload more blobs today.
 */
export async function canUploadBlob(env: Env, teamId: string, currentCount: number): Promise<boolean> {
    const limits = await getTeamLimits(env, teamId);
    return limits.maxBlobsPerDay < 0 || currentCount < limits.maxBlobsPerDay;
}

/**
 * Get the blob TTL for a team's tier.
 */
export async function getBlobTtl(env: Env, teamId: string): Promise<number> {
    const limits = await getTeamLimits(env, teamId);
    return limits.blobTtlHours * 3600;
}

/**
 * Format an upgrade message for tier limit errors.
 */
export function upgradeMessage(limit: string): string {
    return `${limit}. Upgrade at https://envsync.dev/pricing or run: envsync upgrade`;
}
