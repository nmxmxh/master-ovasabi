/**
 * Multi-layer state management for user sessions
 * Handles localStorage clearing while maintaining frontend state
 */

export interface UserState {
  userId: string;
  sessionId: string;
  deviceId: string;
  timestamp: number;
  isTemporary: boolean;
  version: string; // For migration/validation
}

export class StateManager {
  private static instance: StateManager;
  private state: UserState | null = null;
  private readonly VERSION = '1.0.0';
  private readonly STORAGE_KEYS = {
    TEMP: 'temp_user_state',
    PERSISTENT: 'persistent_user_state',
    MIGRATION: 'user_state_migration'
  };

  static getInstance(): StateManager {
    if (!StateManager.instance) {
      StateManager.instance = new StateManager();
    }
    return StateManager.instance;
  }

  /**
   * Initialize state with multi-layer fallback
   */
  async initialize(): Promise<UserState> {
    // 1. Try to get from WASM (highest priority)
    const wasmState = this.getFromWasm();
    if (wasmState) {
      this.state = wasmState;
      return wasmState;
    }

    // 2. Try session storage (survives refresh, cleared on tab close)
    const sessionState = this.getFromSessionStorage();
    if (sessionState) {
      this.state = sessionState;
      return sessionState;
    }

    // 3. Try persistent storage (survives sessions)
    const persistentState = this.getFromPersistentStorage();
    if (persistentState) {
      this.state = persistentState;
      // Copy to session storage for current session
      this.saveToSessionStorage(persistentState);
      return persistentState;
    }

    // 4. Generate new state
    const newState = await this.generateNewState();
    this.state = newState;
    this.saveToSessionStorage(newState);
    this.saveToPersistentStorage(newState);
    return newState;
  }

  /**
   * Get state from WASM (source of truth)
   */
  private getFromWasm(): UserState | null {
    if (typeof window === 'undefined') return null;

    const wasmUserId = (window as any).userID;
    if (!wasmUserId) return null;

    return {
      userId: wasmUserId,
      sessionId: this.getOrGenerateSessionId(),
      deviceId: this.getOrGenerateDeviceId(),
      timestamp: Date.now(),
      isTemporary: false,
      version: this.VERSION
    };
  }

  /**
   * Get state from session storage (survives refresh)
   */
  private getFromSessionStorage(): UserState | null {
    try {
      const stored = sessionStorage.getItem(this.STORAGE_KEYS.TEMP);
      if (!stored) return null;

      const state = JSON.parse(stored);
      return this.validateState(state) ? state : null;
    } catch {
      return null;
    }
  }

  /**
   * Get state from persistent storage
   */
  private getFromPersistentStorage(): UserState | null {
    try {
      const stored = localStorage.getItem(this.STORAGE_KEYS.PERSISTENT);
      if (!stored) return null;

      const state = JSON.parse(stored);
      return this.validateState(state) ? state : null;
    } catch {
      return null;
    }
  }

  /**
   * Generate new user state
   */
  private async generateNewState(): Promise<UserState> {
    const { generateUserID, generateSessionID, generateDeviceID } = await import(
      './wasmIdExtractor'
    );

    return {
      userId: await generateUserID(),
      sessionId: await generateSessionID(),
      deviceId: await generateDeviceID(),
      timestamp: Date.now(),
      isTemporary: true,
      version: this.VERSION
    };
  }

  /**
   * Save to session storage (survives refresh)
   */
  private saveToSessionStorage(state: UserState): void {
    try {
      sessionStorage.setItem(this.STORAGE_KEYS.TEMP, JSON.stringify(state));
    } catch (error) {
      console.warn('Failed to save to session storage:', error);
    }
  }

  /**
   * Save to persistent storage
   */
  private saveToPersistentStorage(state: UserState): void {
    try {
      localStorage.setItem(this.STORAGE_KEYS.PERSISTENT, JSON.stringify(state));
    } catch (error) {
      console.warn('Failed to save to persistent storage:', error);
    }
  }

  /**
   * Clear all storage (for migration/cleanup)
   */
  clearAllStorage(): void {
    try {
      sessionStorage.removeItem(this.STORAGE_KEYS.TEMP);
      localStorage.removeItem(this.STORAGE_KEYS.PERSISTENT);
      localStorage.removeItem(this.STORAGE_KEYS.MIGRATION);
    } catch (error) {
      console.warn('Failed to clear storage:', error);
    }
  }

  /**
   * Clear only persistent storage (keep session state)
   */
  clearPersistentStorage(): void {
    try {
      localStorage.removeItem(this.STORAGE_KEYS.PERSISTENT);
      localStorage.removeItem(this.STORAGE_KEYS.MIGRATION);
    } catch (error) {
      console.warn('Failed to clear persistent storage:', error);
    }
  }

  /**
   * Migrate old state format to new format
   */
  migrateOldState(): void {
    try {
      // Check for old format and migrate
      const oldGuestId = localStorage.getItem('guest_id');
      if (oldGuestId && oldGuestId.length < 32) {
        console.log('Migrating old guest ID format:', oldGuestId);
        localStorage.removeItem('guest_id');
        this.clearAllStorage();
        // Force regeneration on next initialization
      }
    } catch (error) {
      console.warn('Failed to migrate old state:', error);
    }
  }

  /**
   * Get current state
   */
  getState(): UserState | null {
    return this.state;
  }

  /**
   * Update state and persist
   */
  updateState(updates: Partial<UserState>): void {
    if (!this.state) return;

    this.state = { ...this.state, ...updates, timestamp: Date.now() };
    this.saveToSessionStorage(this.state);
    this.saveToPersistentStorage(this.state);
  }

  /**
   * Validate state format
   */
  private validateState(state: any): state is UserState {
    return (
      state &&
      typeof state.userId === 'string' &&
      typeof state.sessionId === 'string' &&
      typeof state.deviceId === 'string' &&
      typeof state.timestamp === 'number' &&
      typeof state.isTemporary === 'boolean' &&
      state.version === this.VERSION
    );
  }

  /**
   * Get or generate session ID
   */
  private getOrGenerateSessionId(): string {
    try {
      let sessionId = sessionStorage.getItem('session_id');
      if (!sessionId) {
        sessionId = Math.random().toString(36).substring(2) + Date.now().toString(36);
        sessionStorage.setItem('session_id', sessionId);
      }
      return sessionId;
    } catch {
      return Math.random().toString(36).substring(2) + Date.now().toString(36);
    }
  }

  /**
   * Get or generate device ID
   */
  private getOrGenerateDeviceId(): string {
    try {
      let deviceId = localStorage.getItem('device_id');
      if (!deviceId) {
        deviceId = Math.random().toString(36).substring(2) + Date.now().toString(36);
        localStorage.setItem('device_id', deviceId);
      }
      return deviceId;
    } catch {
      return Math.random().toString(36).substring(2) + Date.now().toString(36);
    }
  }
}

// Export singleton instance
export const stateManager = StateManager.getInstance();
