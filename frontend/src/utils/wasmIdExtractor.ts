/**
 * WASM ID Extractor
 *
 * This utility extracts IDs from WASM, which is the single source of truth
 * for ID generation across the entire application.
 */

// Type definitions for WASM ID generation functions are declared globally

// Extend Window interface to include WASM functions
declare global {
  interface Window {
    generateUserID?: () => string;
    generateGuestID?: () => string;
    generateSessionID?: () => string;
    generateDeviceID?: () => string;
    generateCampaignID?: () => string;
    generateCorrelationID?: () => string;
  }
}

/**
 * Check if WASM ID generation functions are available
 */
function isWasmIdGeneratorAvailable(): boolean {
  return (
    typeof window !== 'undefined' &&
    typeof window.generateUserID === 'function' &&
    typeof window.generateGuestID === 'function' &&
    typeof window.generateSessionID === 'function' &&
    typeof window.generateDeviceID === 'function' &&
    typeof window.generateCampaignID === 'function' &&
    typeof window.generateCorrelationID === 'function'
  );
}

/**
 * Wait for WASM ID generation functions to be available
 */
async function waitForWasmIdGenerator(timeoutMs: number = 5000): Promise<boolean> {
  const startTime = Date.now();

  while (Date.now() - startTime < timeoutMs) {
    if (isWasmIdGeneratorAvailable()) {
      return true;
    }
    await new Promise(resolve => setTimeout(resolve, 100));
  }

  return false;
}

/**
 * Generate a user ID from WASM
 */
export async function generateUserID(): Promise<string> {
  if (isWasmIdGeneratorAvailable()) {
    return window.generateUserID!();
  }

  // Wait for WASM to be ready
  const isReady = await waitForWasmIdGenerator();
  if (isReady) {
    return window.generateUserID!();
  }

  throw new Error('WASM ID generator not available - WASM may not be loaded');
}

/**
 * Generate a guest ID from WASM
 */
export async function generateGuestID(): Promise<string> {
  if (isWasmIdGeneratorAvailable()) {
    return window.generateGuestID!();
  }

  const isReady = await waitForWasmIdGenerator();
  if (isReady) {
    return window.generateGuestID!();
  }

  throw new Error('WASM ID generator not available - WASM may not be loaded');
}

/**
 * Generate a session ID from WASM
 */
export async function generateSessionID(): Promise<string> {
  if (isWasmIdGeneratorAvailable()) {
    return window.generateSessionID!();
  }

  const isReady = await waitForWasmIdGenerator();
  if (isReady) {
    return window.generateSessionID!();
  }

  throw new Error('WASM ID generator not available - WASM may not be loaded');
}

/**
 * Generate a device ID from WASM
 */
export async function generateDeviceID(): Promise<string> {
  if (isWasmIdGeneratorAvailable()) {
    return window.generateDeviceID!();
  }

  const isReady = await waitForWasmIdGenerator();
  if (isReady) {
    return window.generateDeviceID!();
  }

  throw new Error('WASM ID generator not available - WASM may not be loaded');
}

/**
 * Generate a campaign ID from WASM
 */
export async function generateCampaignID(): Promise<string> {
  if (isWasmIdGeneratorAvailable()) {
    return window.generateCampaignID!();
  }

  const isReady = await waitForWasmIdGenerator();
  if (isReady) {
    return window.generateCampaignID!();
  }

  throw new Error('WASM ID generator not available - WASM may not be loaded');
}

/**
 * Generate a correlation ID from WASM
 */
export async function generateCorrelationID(): Promise<string> {
  if (isWasmIdGeneratorAvailable()) {
    return window.generateCorrelationID!();
  }

  const isReady = await waitForWasmIdGenerator();
  if (isReady) {
    return window.generateCorrelationID!();
  }

  throw new Error('WASM ID generator not available - WASM may not be loaded');
}

/**
 * Synchronous version that returns a fallback if WASM is not ready
 * Use this only when you need immediate results and can handle fallbacks
 */
export function generateUserIDSync(): string {
  if (isWasmIdGeneratorAvailable()) {
    return window.generateUserID!();
  }
  return 'user_fallback_' + Date.now();
}

export function generateGuestIDSync(): string {
  if (isWasmIdGeneratorAvailable()) {
    return window.generateGuestID!();
  }
  return 'guest_fallback_' + Date.now();
}

export function generateSessionIDSync(): string {
  if (isWasmIdGeneratorAvailable()) {
    return window.generateSessionID!();
  }
  return 'session_fallback_' + Date.now();
}

export function generateDeviceIDSync(): string {
  if (isWasmIdGeneratorAvailable()) {
    return window.generateDeviceID!();
  }
  return 'device_fallback_' + Date.now();
}

export function generateCampaignIDSync(): string {
  if (isWasmIdGeneratorAvailable()) {
    return window.generateCampaignID!();
  }
  return 'campaign_fallback_' + Date.now();
}

export function generateCorrelationIDSync(): string {
  if (isWasmIdGeneratorAvailable()) {
    return window.generateCorrelationID!();
  }
  return 'corr_fallback_' + Date.now();
}

/**
 * Check if WASM ID generator is ready
 */
export function isWasmIdGeneratorReady(): boolean {
  return isWasmIdGeneratorAvailable();
}
