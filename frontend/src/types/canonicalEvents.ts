/**
 * Canonical Event Types
 *
 * These types match the protobuf definitions in api/protos/common/v1/metadata.proto
 * and ensure type safety across the entire event system.
 */

export interface CanonicalEventEnvelope {
  // Core event fields
  type: string;
  correlation_id: string;
  timestamp: string; // ISO 8601 timestamp
  version: string;
  environment: string;
  source: 'frontend' | 'backend' | 'wasm';

  // Use existing protobuf structures
  metadata: CanonicalMetadata;
  payload?: CanonicalPayload;
}

export interface CanonicalMetadata {
  // Global context - always present
  global_context: GlobalContext;

  // Envelope versioning
  envelope_version: string;
  environment: string;

  // Core protobuf fields
  scheduling?: Record<string, any>;
  features: string[];
  custom_rules?: Record<string, any>;
  audit?: Record<string, any>;
  tags: string[];
  ServiceSpecific: Record<string, any>;
  knowledge_graph?: KnowledgeGraph;
  taxation?: TieredTax;
  owner?: OwnerMetadata;
  referral?: ReferralMetadata;
  versioning?: Record<string, any>;

  // Intelligence system fields
  ai_confidence?: number;
  embedding_id?: string;
  categories?: string[];
  last_accessed?: string; // ISO 8601 timestamp
  nexus_channel?: string;
  source_uri?: string;

  // Scheduler optimizations
  scheduler?: SchedulerConfig;
}

export interface GlobalContext {
  user_id: string;
  campaign_id: string;
  correlation_id: string;
  session_id: string;
  device_id: string;
  source: 'frontend' | 'backend' | 'wasm';
}

export interface CanonicalPayload {
  data: Record<string, any>;
  error?: EventError;
}

export interface EventError {
  code: string;
  message: string;
  details?: any;
}

// Protobuf message types (matching the proto definitions)
export interface KnowledgeGraph {
  id: string;
  name: string;
  nodes: string[];
  edges: string[];
  description: string;
}

export interface TieredTax {
  min_projects: number;
  max_projects: number;
  percentage: number;
}

export interface OwnerMetadata {
  id: string;
  wallet: string;
  uri: string;
}

export interface ReferralMetadata {
  id: string;
  wallet: string;
  uri: string;
}

export interface SchedulerConfig {
  is_ephemeral: boolean;
  expiry?: string; // ISO 8601 timestamp
  job_dependencies: string[];
  retention_policy: string;
}

// Validation functions
export function validateCanonicalEventEnvelope(envelope: any): envelope is CanonicalEventEnvelope {
  if (!envelope || typeof envelope !== 'object') return false;

  // Check required fields
  if (!envelope.type || typeof envelope.type !== 'string') return false;
  if (!envelope.correlation_id || typeof envelope.correlation_id !== 'string') return false;
  if (!envelope.timestamp || typeof envelope.timestamp !== 'string') return false;
  if (!envelope.version || typeof envelope.version !== 'string') return false;
  if (!envelope.environment || typeof envelope.environment !== 'string') return false;
  if (!envelope.source || typeof envelope.source !== 'string') return false;

  // Check metadata structure
  if (!envelope.metadata || typeof envelope.metadata !== 'object') return false;
  if (!envelope.metadata.global_context || typeof envelope.metadata.global_context !== 'object')
    return false;

  const global = envelope.metadata.global_context;
  if (!global.user_id || !global.campaign_id || !global.correlation_id) return false;
  if (!global.session_id || !global.device_id || !global.source) return false;

  // Validate timestamp format
  if (isNaN(Date.parse(envelope.timestamp))) return false;

  return true;
}

export function validateGlobalContext(context: any): context is GlobalContext {
  if (!context || typeof context !== 'object') return false;

  return (
    typeof context.user_id === 'string' &&
    typeof context.campaign_id === 'string' &&
    typeof context.correlation_id === 'string' &&
    typeof context.session_id === 'string' &&
    typeof context.device_id === 'string' &&
    typeof context.source === 'string' &&
    ['frontend', 'backend', 'wasm'].includes(context.source)
  );
}

// Helper functions
export function createCanonicalEventEnvelope(
  type: string,
  userID: string,
  campaignID: string,
  correlationID: string,
  payload?: Record<string, any>,
  serviceSpecific: Record<string, any> = {}
): CanonicalEventEnvelope {
  const now = new Date().toISOString();

  return {
    type,
    correlation_id: correlationID,
    timestamp: now,
    version: '1.0.0',
    environment: process.env.NODE_ENV || 'development',
    source: 'frontend',

    payload: payload
      ? {
          data: payload
        }
      : undefined,

    metadata: {
      global_context: {
        user_id: userID,
        campaign_id: campaignID,
        correlation_id: correlationID,
        session_id: generateSessionID(),
        device_id: generateDeviceID(),
        source: 'frontend'
      },
      envelope_version: '1.0.0',
      environment: process.env.NODE_ENV || 'development',
      ServiceSpecific: serviceSpecific,
      features: [],
      tags: [],
      audit: {
        created_at: now,
        created_by: userID
      }
    }
  };
}

export function extractExpectedSuccessType(eventType: string): string {
  if (eventType.endsWith(':request')) {
    return eventType.replace(/:request$/, ':success');
  }
  if (eventType.endsWith(':requested')) {
    return eventType.replace(/:requested$/, ':success');
  }
  return eventType;
}

// Utility functions
import { generateSessionId, generateDeviceId } from '../utils/cryptoIds';

function generateSessionID(): string {
  return generateSessionId();
}

function generateDeviceID(): string {
  return generateDeviceId();
}

// Type guards for runtime type checking
export function isCanonicalEventEnvelope(obj: any): obj is CanonicalEventEnvelope {
  return validateCanonicalEventEnvelope(obj);
}

export function isGlobalContext(obj: any): obj is GlobalContext {
  return validateGlobalContext(obj);
}

// Helper function to transform flat metadata to canonical format
export function transformToCanonicalMetadata(
  flatMetadata: any,
  correlationId: string
): CanonicalMetadata {
  // Extract campaign features from campaign metadata
  const campaignFeatures = flatMetadata.campaign?.features || [];
  const campaignTags = flatMetadata.campaign?.tags || [];

  return {
    global_context: {
      user_id: flatMetadata.user?.userId || flatMetadata.session?.guestId || 'anonymous',
      campaign_id: String(flatMetadata.campaign?.campaignId || flatMetadata.campaign?.slug || '0'),
      correlation_id: correlationId,
      session_id: flatMetadata.session?.sessionId || 'current_session',
      device_id: flatMetadata.device?.deviceId || 'current_device',
      source: 'frontend'
    },
    envelope_version: '1.0.0',
    environment: process.env.NODE_ENV || 'development',
    ServiceSpecific: {
      campaign: flatMetadata.campaign,
      user: flatMetadata.user,
      device: flatMetadata.device,
      session: flatMetadata.session
    },
    features: campaignFeatures,
    tags: campaignTags,
    audit: {
      created_at: new Date().toISOString(),
      created_by: flatMetadata.user?.userId || 'anonymous'
    }
  };
}
