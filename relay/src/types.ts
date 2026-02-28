// EnvSync Relay — Shared Types

export interface Env {
    ENVSYNC_DATA: KVNamespace;
    SIGNALING_ROOM: DurableObjectNamespace;
    ENVIRONMENT: string;
    MAX_BLOB_SIZE: string;
    INVITE_TTL_HOURS: string;
    BLOB_TTL_HOURS: string;
}

// --- Invite Types ---

export interface Invite {
    /** SHA-256 hash of the 6-word mnemonic token */
    token_hash: string;
    /** Team ID this invite belongs to */
    team_id: string;
    /** GitHub username of the inviter */
    inviter: string;
    /** Fingerprint of the inviter's SSH key */
    inviter_fingerprint: string;
    /** GitHub username of the invitee */
    invitee: string;
    /** Expected fingerprint of the invitee (from GitHub keys) */
    expected_fingerprint: string;
    /** Unix timestamp when invite was created */
    created_at: number;
    /** Unix timestamp when invite expires */
    expires_at: number;
    /** Whether the invite has been consumed */
    consumed: boolean;
}

// --- Blob Types ---

export interface BlobMetadata {
    /** Unique blob ID */
    blob_id: string;
    /** Team ID */
    team_id: string;
    /** Fingerprint of the sender */
    sender_fingerprint: string;
    /** Fingerprint of the intended recipient */
    recipient_fingerprint: string;
    /** Ephemeral public key for ECDH (base64) */
    ephemeral_public_key: string;
    /** Size in bytes */
    size: number;
    /** Unix timestamp of upload */
    uploaded_at: number;
    /** Unix timestamp of expiry */
    expires_at: number;
    /** Original filename */
    filename: string;
}

// --- Team Types ---

export interface TeamMember {
    /** GitHub username */
    username: string;
    /** SSH key fingerprint */
    fingerprint: string;
    /** Ed25519 public key (base64) for signature verification */
    public_key: string;
    /** Role: owner or member */
    role: 'owner' | 'member';
    /** Unix timestamp when added */
    added_at: number;
}

export interface Team {
    /** Team ID */
    id: string;
    /** Team name */
    name: string;
    /** Members list */
    members: TeamMember[];
    /** Unix timestamp of creation */
    created_at: number;
}

// --- Signaling Types ---

export interface SignalMessage {
    type: 'offer' | 'answer' | 'candidate';
    sender_fingerprint: string;
    /** Public IP address */
    public_ip: string;
    /** Public port */
    public_port: number;
    /** Local IP address */
    local_ip: string;
    /** Local port */
    local_port: number;
    /** NAT type: full-cone, restricted, symmetric, unknown */
    nat_type: string;
}

// --- Auth Types ---

export interface AuthInfo {
    fingerprint: string;
    timestamp: number;
    verified: boolean;
}
