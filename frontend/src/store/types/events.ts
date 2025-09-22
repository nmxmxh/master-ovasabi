// Event-related type definitions
export interface EventEnvelope {
  type: string; // {service}:{action}:v{version}:{state}
  correlation_id: string;
  timestamp: string; // ISO string with timezone
  version: string;
  environment: string;
  source: 'frontend' | 'backend' | 'wasm';
  payload?: any;
  metadata: any; // Will be imported from metadata types
}

export interface PendingRequestEntry {
  expectedEventType: string;
  eventType: string;
  timestamp: number;
  resolve: (event: EventEnvelope) => void;
  reject?: (reason?: any) => void;
}

export interface EventState {
  events: EventEnvelope[];
  eventStates: Record<string, string>; // eventType -> current state
  eventPayloads: Record<string, any>; // eventType -> payload for proactive state
  queuedMessages: EventEnvelope[];
  pendingRequests: Record<string, PendingRequestEntry>;
}
