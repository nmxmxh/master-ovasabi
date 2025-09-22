/**
 * Secure ID Generation Utilities
 *
 * This module provides cryptographically secure ID generation functions
 * using the Web Crypto API for better security and uniqueness guarantees.
 */

/**
 * Generate a cryptographically secure random ID
 * @param length - Length of the ID (default: 32)
 * @param prefix - Optional prefix for the ID
 * @returns Secure random ID string
 */
export function generateSecureId(length: number = 32, prefix?: string): string {
  if (typeof window !== 'undefined' && window.crypto && window.crypto.getRandomValues) {
    // Use Web Crypto API for secure random generation
    const array = new Uint8Array(Math.ceil(length / 2));
    window.crypto.getRandomValues(array);

    // Convert to hex string
    const hexString = Array.from(array)
      .map(byte => byte.toString(16).padStart(2, '0'))
      .join('');

    const id = hexString.substring(0, length);
    return prefix ? `${prefix}_${id}` : id;
  }

  // Fallback for environments without crypto support
  console.warn('[CryptoIds] Web Crypto API not available, using fallback method');
  const fallbackId =
    Math.random()
      .toString(36)
      .substring(2, 2 + length) +
    Math.random()
      .toString(36)
      .substring(2, 2 + length);
  return prefix ? `${prefix}_${fallbackId}` : fallbackId;
}

/**
 * Generate a secure UUID v4 using crypto
 * @returns RFC4122 compliant UUID v4
 */
export function generateSecureUUID(): string {
  if (typeof window !== 'undefined' && window.crypto && window.crypto.getRandomValues) {
    // Generate 16 random bytes
    const array = new Uint8Array(16);
    window.crypto.getRandomValues(array);

    // Set version (4) and variant bits
    array[6] = (array[6] & 0x0f) | 0x40; // Version 4
    array[8] = (array[8] & 0x3f) | 0x80; // Variant bits

    // Convert to UUID string format
    const hex = Array.from(array)
      .map(byte => byte.toString(16).padStart(2, '0'))
      .join('');

    return [
      hex.substring(0, 8),
      hex.substring(8, 12),
      hex.substring(12, 16),
      hex.substring(16, 20),
      hex.substring(20, 32)
    ].join('-');
  }

  // Fallback UUID generation
  console.warn('[CryptoIds] Web Crypto API not available, using fallback UUID generation');
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function (c) {
    const r = (Math.random() * 16) | 0;
    const v = c === 'x' ? r : (r & 0x3) | 0x8;
    return v.toString(16);
  });
}

/**
 * Generate a secure correlation ID for event tracking
 * @returns Secure correlation ID
 */
export function generateCorrelationId(): string {
  return generateSecureId(24, 'corr');
}

/**
 * Generate a secure device ID
 * @returns Secure device ID
 */
export function generateDeviceId(): string {
  return generateSecureId(32, 'device');
}

/**
 * Generate a secure session ID
 * @returns Secure session ID
 */
export function generateSessionId(): string {
  return generateSecureId(32, 'session');
}

/**
 * Generate a secure user/guest ID
 * @returns Secure user ID
 */
export function generateUserId(): string {
  return generateSecureId(32, 'user');
}

/**
 * Generate a secure task ID for compute operations
 * @returns Secure task ID
 */
export function generateTaskId(): string {
  return generateSecureId(24, 'task');
}

/**
 * Generate a secure campaign ID
 * @returns Secure campaign ID
 */
export function generateCampaignId(): string {
  return generateSecureId(24, 'campaign');
}

/**
 * Generate a secure hash from input data
 * @param data - Data to hash
 * @param algorithm - Hash algorithm (default: 'SHA-256')
 * @returns Promise resolving to hex hash string
 */
export async function generateSecureHash(
  data: string | ArrayBuffer,
  algorithm: string = 'SHA-256'
): Promise<string> {
  if (typeof window !== 'undefined' && window.crypto && window.crypto.subtle) {
    try {
      const encoder = new TextEncoder();
      const dataBuffer = typeof data === 'string' ? encoder.encode(data) : data;
      const hashBuffer = await window.crypto.subtle.digest(algorithm, dataBuffer);
      const hashArray = new Uint8Array(hashBuffer);
      return Array.from(hashArray)
        .map(byte => byte.toString(16).padStart(2, '0'))
        .join('');
    } catch (error) {
      console.warn('[CryptoIds] Hash generation failed:', error);
    }
  }

  // Fallback: simple hash using built-in functions
  console.warn('[CryptoIds] Web Crypto Subtle API not available, using fallback hash');
  const str = typeof data === 'string' ? data : new TextDecoder().decode(data);
  let hash = 0;
  for (let i = 0; i < str.length; i++) {
    const char = str.charCodeAt(i);
    hash = (hash << 5) - hash + char;
    hash = hash & hash; // Convert to 32-bit integer
  }
  return Math.abs(hash).toString(16);
}

/**
 * Generate a secure nonce for cryptographic operations
 * @param length - Length of nonce in bytes (default: 16)
 * @returns Secure nonce as hex string
 */
export function generateNonce(length: number = 16): string {
  return generateSecureId(length * 2); // *2 because hex encoding doubles the length
}

/**
 * Validate if an ID appears to be securely generated
 * @param id - ID to validate
 * @returns True if ID appears secure
 */
export function isSecureId(id: string): boolean {
  // Check if ID contains only hex characters and is of reasonable length
  const hexPattern = /^[a-f0-9_]+$/i;
  return hexPattern.test(id) && id.length >= 16;
}

/**
 * Generate a secure timestamp-based ID with crypto entropy
 * @param prefix - Optional prefix
 * @returns Secure timestamp-based ID
 */
export function generateTimestampId(prefix?: string): string {
  const timestamp = Date.now().toString(36);
  const entropy = generateSecureId(16);
  const id = `${timestamp}_${entropy}`;
  return prefix ? `${prefix}_${id}` : id;
}
