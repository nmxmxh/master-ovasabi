// Metadata-related type definitions
import type { Campaign } from './campaign';

export interface UserMetadata {
  userId?: string;
  username?: string;
  privileges?: string[];
  referralCode?: string;
}

export interface DeviceMetadata {
  deviceId: string;
  userAgent?: string;
  userAgentData?: {
    brands?: Array<{ brand: string; version: string }>;
    mobile?: boolean;
    platform?: string;
  };
  language?: string;
  timezone?: string;
  consentGiven: boolean;
  gdprConsentTimestamp?: string; // ISO string with timezone
  gdprConsentRequired?: boolean;
  // Security and tracking information
  ipAddress?: string;
  ipLocation?: {
    country?: string;
    region?: string;
    city?: string;
    latitude?: number;
    longitude?: number;
  };
  securityFlags?: {
    isBot?: boolean;
    isVPN?: boolean;
    isProxy?: boolean;
    riskScore?: number;
    suspiciousActivity?: boolean;
  };
  // GPU and performance information
  gpuCapabilities?: any; // Will be populated by WASM GPU Bridge
  wasmGPUBridge?: {
    initialized: boolean;
    backend: string;
    workerCount: number;
    version: string;
  };
  gpuDetectedAt?: string;
  [key: string]: any; // Allow additional device properties
}

export interface SessionMetadata {
  sessionId: string;
  guestId?: string;
  authenticated?: boolean;
}

export interface Metadata {
  campaign: Campaign;
  user: UserMetadata;
  device: DeviceMetadata;
  session: SessionMetadata;
  correlation_id?: string; // Backend expects this field
  [key: string]: any;
}
