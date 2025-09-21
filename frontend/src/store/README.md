# 🏗️ Modular Store Architecture

This directory contains a refactored, modular store architecture that replaces the monolithic
`global.ts` file with focused, single-responsibility stores.

## 📁 Structure

```
store/
├── types/                    # Type definitions
│   ├── index.ts             # Re-exports all types
│   ├── events.ts            # Event-related types
│   ├── campaign.ts          # Campaign-related types
│   ├── connection.ts        # Connection-related types
│   └── metadata.ts          # Metadata-related types
├── stores/                  # Individual Zustand stores
│   ├── connectionStore.ts   # WebSocket/WASM connection management
│   ├── eventStore.ts        # Event handling and queuing
│   ├── campaignStore.ts     # Campaign state and operations
│   └── metadataStore.ts     # User/device/session metadata
├── hooks/                   # React hooks for store access
│   ├── useConnection.ts     # Connection status hooks
│   ├── useEvents.ts         # Event handling hooks
│   ├── useCampaign.ts       # Campaign management hooks
│   └── useMetadata.ts       # Metadata hooks
├── index.ts                 # Main exports
├── MIGRATION_GUIDE.md       # Migration guide from old store
└── README.md               # This file
```

## 🎯 Design Principles

### 1. **Separation of Concerns**

Each store handles one specific responsibility:

- **Connection Store**: WebSocket connections, WASM readiness, media streaming
- **Event Store**: Event emission, history, state management
- **Campaign Store**: Campaign operations, state management
- **Metadata Store**: User/device/session information

### 2. **Single Responsibility**

Each store has a clear, focused purpose:

- No store handles multiple unrelated concerns
- Clear boundaries between different domains
- Easier to test and maintain

### 3. **Type Safety**

Strong TypeScript support:

- Dedicated type files for each domain
- Proper interfaces and type definitions
- Better IDE support and error catching

### 4. **Performance**

Optimized for React:

- Smaller stores with less re-rendering
- Focused selectors and hooks
- Better memoization opportunities

## 🚀 Usage

### Basic Usage

```typescript
import { useConnectionStatus, useEventHistory, useCampaignState, useMetadata } from './store';

function MyComponent() {
  const { connected, isConnected } = useConnectionStatus();
  const events = useEventHistory('campaign:update:v1:success');
  const { state: campaignState } = useCampaignState();
  const { metadata } = useMetadata();

  // Component logic
}
```

### Advanced Usage

```typescript
import { useEventStore, useCampaignStore } from './store';

function AdvancedComponent() {
  const emitEvent = useEventStore(state => state.emitEvent);
  const updateCampaign = useCampaignStore(state => state.updateCampaign);

  // Direct store access for advanced use cases
}
```

## 🔧 Store Details

### Connection Store

- **Purpose**: Manages WebSocket connections and WASM readiness
- **State**: Connection status, WASM functions, media streaming
- **Key Hooks**: `useConnectionStatus()`, `useMediaStreamingState()`

### Event Store

- **Purpose**: Handles event emission and management
- **State**: Event history, event states, pending requests
- **Key Hooks**: `useEmitEvent()`, `useEventHistory()`, `useEventState()`

### Campaign Store

- **Purpose**: Manages campaign state and operations
- **State**: Current campaign, campaign operations
- **Key Hooks**: `useCampaignState()`, `useCampaignUpdates()`

### Metadata Store

- **Purpose**: Stores user/device/session information
- **State**: User info, device info, session info
- **Key Hooks**: `useMetadata()`, `useUserMetadata()`, `useDeviceMetadata()`

## 🎨 Benefits

1. **Maintainability**: Easier to understand and modify individual concerns
2. **Testability**: Each store can be tested in isolation
3. **Performance**: Smaller stores with focused re-rendering
4. **Type Safety**: Better TypeScript support and error catching
5. **Developer Experience**: Clearer code organization and better IDE support

## 🔄 Migration

See [MIGRATION_GUIDE.md](./MIGRATION_GUIDE.md) for detailed migration instructions from the old
`global.ts` store.

## 🧪 Testing

Each store can be tested independently:

```typescript
import { useConnectionStore } from './stores/connectionStore';

// Test connection store in isolation
test('connection store updates state correctly', () => {
  const { result } = renderHook(() => useConnectionStore());

  act(() => {
    result.current.setConnectionState({ connected: true });
  });

  expect(result.current.connected).toBe(true);
});
```

## 📝 Best Practices

1. **Use Hooks**: Prefer the provided hooks over direct store access
2. **Single Responsibility**: Don't mix concerns in components
3. **Type Safety**: Always use proper TypeScript types
4. **Performance**: Use focused selectors to minimize re-renders
5. **Testing**: Test stores in isolation when possible

## 🆘 Troubleshooting

### Common Issues

1. **Import Errors**: Make sure to import from the correct store module
2. **Type Errors**: Check that you're using the correct types from `./types`
3. **Performance Issues**: Use focused selectors and avoid unnecessary re-renders

### Getting Help

1. Check the individual store implementations
2. Look at the type definitions in `./types`
3. Review the migration guide for common patterns
4. Test with individual stores first before combining them
