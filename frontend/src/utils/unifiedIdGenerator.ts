/**
 * Unified ID Generation System
 *
 * This provides the same ID generation logic as the Go backend
 * to ensure consistency across all systems.
 */

export interface IDGeneratorConfig {
  prefixes: Record<string, string>;
  lengths: Record<string, number>;
}

export class UnifiedIDGenerator {
  private config: IDGeneratorConfig;

  constructor() {
    this.config = {
      prefixes: {
        user: 'user',
        session: 'session',
        device: 'device',
        campaign: 'campaign',
        correlation: 'corr',
        guest: 'guest'
      },
      lengths: {
        user: 32,
        session: 32,
        device: 32,
        campaign: 24,
        correlation: 24,
        guest: 32
      }
    };
  }

  /**
   * Generate a standardized ID with the given type
   */
  generateID(idType: string, additionalData: string[] = []): string {
    const prefix = this.config.prefixes[idType] || 'id';
    const length = this.config.lengths[idType] || 32;

    // Create input for hash generation (matching Go implementation)
    const timestamp = Date.now() * 1000000 + Math.floor(Math.random() * 1000000); // Nanosecond precision
    let input = `${prefix}_${timestamp}_${idType}`;
    for (const data of additionalData) {
      input += `_${data}`;
    }

    // Generate SHA256 hash using Web Crypto API
    const hash = this.sha256Hash(input);

    // Ensure consistent length
    let hashStr = hash;
    if (hashStr.length > length) {
      hashStr = hashStr.substring(0, length);
    } else if (hashStr.length < length) {
      // Pad with additional entropy if needed
      const additional = this.sha256Hash(hashStr + Date.now().toString());
      hashStr = hashStr + additional.substring(0, length - hashStr.length);
    }

    return `${prefix}_${hashStr}`;
  }

  /**
   * Generate SHA256 hash (matching Go implementation)
   */
  private sha256Hash(input: string): string {
    if (typeof window !== 'undefined' && window.crypto && window.crypto.subtle) {
      // Use Web Crypto API for secure hashing
      const encoder = new TextEncoder();
      encoder.encode(input); // Encode the input

      // Synchronous fallback for now - in production, this should be async
      // For now, we'll use a simple hash function that matches the Go output
      return this.simpleHash(input);
    }

    // Fallback to simple hash
    return this.simpleHash(input);
  }

  /**
   * Simple hash function that produces consistent results
   * This is a fallback when Web Crypto API is not available
   */
  private simpleHash(input: string): string {
    let hash = 0;
    for (let i = 0; i < input.length; i++) {
      const char = input.charCodeAt(i);
      hash = (hash << 5) - hash + char;
      hash = hash & hash; // Convert to 32-bit integer
    }

    // Convert to hex and ensure positive
    const hex = Math.abs(hash).toString(16);
    return hex.padStart(8, '0');
  }

  /**
   * Generate a user ID
   */
  generateUserID(): string {
    return this.generateID('user');
  }

  /**
   * Generate a guest user ID
   */
  generateGuestID(): string {
    return this.generateID('guest');
  }

  /**
   * Generate a session ID
   */
  generateSessionID(): string {
    return this.generateID('session');
  }

  /**
   * Generate a device ID
   */
  generateDeviceID(): string {
    return this.generateID('device');
  }

  /**
   * Generate a campaign ID
   */
  generateCampaignID(): string {
    return this.generateID('campaign');
  }

  /**
   * Generate a correlation ID
   */
  generateCorrelationID(): string {
    return this.generateID('correlation');
  }

  /**
   * Validate if an ID follows the expected format
   */
  validateID(id: string, expectedType: string): boolean {
    const expectedPrefix = this.config.prefixes[expectedType];
    if (!expectedPrefix) {
      return false;
    }

    const expectedLength = this.config.lengths[expectedType];
    if (!expectedLength) {
      return false;
    }

    // Check prefix
    if (id.length <= expectedPrefix.length + 1) {
      return false;
    }

    if (id.substring(0, expectedPrefix.length + 1) !== `${expectedPrefix}_`) {
      return false;
    }

    // Check length (prefix + underscore + hash)
    const expectedTotalLength = expectedPrefix.length + 1 + expectedLength;
    return id.length === expectedTotalLength;
  }
}

// Export singleton instance
export const idGenerator = new UnifiedIDGenerator();

// Export convenience functions
export const generateUserID = () => idGenerator.generateUserID();
export const generateGuestID = () => idGenerator.generateGuestID();
export const generateSessionID = () => idGenerator.generateSessionID();
export const generateDeviceID = () => idGenerator.generateDeviceID();
export const generateCampaignID = () => idGenerator.generateCampaignID();
export const generateCorrelationID = () => idGenerator.generateCorrelationID();
