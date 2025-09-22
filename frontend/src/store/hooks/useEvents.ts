import { useEventStore } from '../stores/eventStore';
import { useMemo } from 'react';

// Event emission hook
export function useEmitEvent() {
  return useEventStore(state => state.emitEvent);
}

// Event history hook
export function useEventHistory(eventType?: string, limit?: number) {
  const events = useEventStore(state => state.events);

  return useMemo(() => {
    let filteredEvents = events;
    if (eventType) {
      filteredEvents = events.filter(event => event.type === eventType);
    }

    if (limit) {
      filteredEvents = filteredEvents.slice(-limit);
    }

    return filteredEvents;
  }, [events, eventType, limit]);
}

// Event state hook
export function useEventState(eventType: string) {
  return useEventStore(state => state.getCurrentState(eventType));
}

// Latest event hook
export function useLatestEvent(eventType?: string) {
  const events = useEventStore(state => state.events);

  return useMemo(() => {
    if (eventType) {
      const filteredEvents = events.filter(event => event.type === eventType);
      return filteredEvents.length > 0 ? filteredEvents[filteredEvents.length - 1] : undefined;
    }
    return events.length > 0 ? events[events.length - 1] : undefined;
  }, [events, eventType]);
}

// Events by type hook
export function useEventsByType(eventType: string) {
  const events = useEventStore(state => state.events);

  return useMemo(() => {
    return events.filter(event => event.type === eventType);
  }, [events, eventType]);
}

// Event store actions hook
export function useEventActions() {
  const { emitEvent, updateEventState, handleWasmMessage, processQueuedMessages, clearHistory } =
    useEventStore();

  return {
    emitEvent,
    updateEventState,
    handleWasmMessage,
    processQueuedMessages,
    clearHistory
  };
}
