import { create } from 'zustand';
import { devtools } from 'zustand/middleware';
import type { ComputeWorker, ComputeTask, ComputeMetrics } from '../types/compute';
import { useEventStore } from './eventStore';

// --- Helper functions to extract data from events ---

const getEventPayload = (event: any) => {
  // Support multiple shapes: CustomEvent(detail), { payload: { data } }, or direct payload
  const fromDetail = event?.detail;
  const fromPayloadData = event?.payload?.data;
  const fromPayload = event?.payload;
  return fromDetail ?? fromPayloadData ?? fromPayload ?? event ?? null;
};

const getGlobalContext = (event: any) => {
  return event?.metadata?.global_context || event?.metadata?.globalContext || {};
};

const normalizeBoolean = (value: any): boolean => {
  return value === true || value === 'true';
};

const mapStatusToWorkerStatus = (type: string, payloadStatus?: string): ComputeWorker['status'] => {
  const t = type || '';
  const s = (payloadStatus || '').toUpperCase();
  if (s === 'REGISTERED' || s === 'ACTIVE') return 'active';
  if (s === 'INACTIVE') return 'inactive';
  if (s === 'OVERLOADED') return 'overloaded';
  if (t.includes('success') || t.includes('update')) return 'active';
  return 'inactive';
};

const extractWorkerFromEvent = (event: any): ComputeWorker | null => {
  try {
    const payload = getEventPayload(event) || {};
    const globalCtx = getGlobalContext(event);
    const src =
      payload.worker_id ||
      payload.workerId ||
      globalCtx?.device_id ||
      globalCtx?.deviceId ||
      globalCtx?.source;
    if (!src) return null;

    const gpu = payload.gpu || {};
    const status = mapStatusToWorkerStatus(event?.type ?? '', payload.status);

    return {
      id: String(src),
      cpuCores: Number(payload.cpu_cores ?? payload.cpuCores ?? 0),
      memoryMb: Number(payload.memory_mb ?? payload.memoryMb ?? 0),
      wasm: normalizeBoolean(payload.wasm),
      threads: normalizeBoolean(payload.threads),
      simd: normalizeBoolean(payload.simd),
      webgpu: normalizeBoolean(payload.webgpu),
      gpuBackend: gpu.backend || undefined,
      gpuFeatures: Array.isArray(gpu.features) ? gpu.features : [],
      status,
      currentLoad: typeof payload.current_load === 'number' ? payload.current_load : payload.currentLoad ?? 0
    };
  } catch {
    return null;
  }
};

const extractTaskFromEvent = (event: any): ComputeTask | null => {
  try {
    const payload = getEventPayload(event);
    if (!payload) return null;

    let status: ComputeTask['status'] = payload.status || 'pending';
    if (!payload.status) {
      if (event.type?.includes('success')) status = 'completed';
      else if (event.type?.includes('failed')) status = 'failed';
      else if (event.type?.includes('assigned')) status = 'assigned';
      else if (event.type?.includes('accepted')) status = 'assigned';
      else if (event.type?.includes('progress')) status = 'in-progress';
      else if (event.type?.includes('requested')) status = 'pending';
    }

    let assignedWorkerIds: string[] | undefined;
    if (payload.worker_id) {
      assignedWorkerIds = [payload.worker_id];
    } else if (payload.workerId) {
      assignedWorkerIds = [payload.workerId];
    } else if (Array.isArray(payload.assigned_worker_ids)) {
      assignedWorkerIds = payload.assigned_worker_ids;
    } else if (Array.isArray(payload.assignedWorkerIds)) {
      assignedWorkerIds = payload.assignedWorkerIds;
    }

    return {
      id: payload.task_id || payload.taskId || event.id || `task-${Date.now()}`,
      status: status,
      requirements: payload.requirements || {},
      assignedWorkerIds: assignedWorkerIds,
      progress: payload.pct || payload.progress || payload.percentage || 0,
      result: payload.outputs || payload.result,
      error: payload.reason || payload.error
    };
  } catch (error) {
    console.error('Failed to extract task from event:', error);
    return null;
  }
};

// --- Zustand Store Definition ---

interface ComputeState {
  workers: Map<string, ComputeWorker>;
  tasks: Map<string, ComputeTask>;
  metrics: ComputeMetrics | null;
  isComputeEnabled: boolean;
  userId: string | null;
}

interface ComputeStore extends ComputeState {
  setUserId: (userId: string) => void;
  setComputeEnabled: (isEnabled: boolean) => void;
  reset: () => void;
  // Public actions for event handling, to be called from eventStore
  handleWorkerUpdate: (event: any) => void;
  handleTaskUpdate: (event: any) => void;
  handleMetricsUpdate: (event: any) => void;
}

const initialState: ComputeState = {
  workers: new Map(),
  tasks: new Map(),
  metrics: null,
  isComputeEnabled: false,
  userId: null
};

const workerUpdateHandler = (state: ComputeState, event: any) => {
  const worker = extractWorkerFromEvent(event);
  if (worker) {
    const existingWorker = state.workers.get(worker.id);
    // Perform a shallow comparison to avoid unnecessary updates
    if (
      existingWorker &&
      existingWorker.cpuCores === worker.cpuCores &&
      existingWorker.memoryMb === worker.memoryMb &&
      existingWorker.wasm === worker.wasm &&
      existingWorker.threads === worker.threads &&
      existingWorker.simd === worker.simd &&
      existingWorker.webgpu === worker.webgpu &&
      existingWorker.gpuBackend === worker.gpuBackend &&
      existingWorker.status === worker.status &&
      existingWorker.currentLoad === worker.currentLoad &&
      // Deep array comparison for gpuFeatures
      ((!existingWorker.gpuFeatures && !worker.gpuFeatures) ||
        (existingWorker.gpuFeatures &&
          worker.gpuFeatures &&
          existingWorker.gpuFeatures.length === worker.gpuFeatures.length &&
          existingWorker.gpuFeatures.every(
            (feature, i) => worker.gpuFeatures && feature === worker.gpuFeatures[i]
          )))
    ) {
      return state; // No change, return current state to prevent re-render
    }
    const newWorkers = new Map(state.workers);
    newWorkers.set(worker.id, worker);
    return { workers: newWorkers };
  }
  return state;
};

const taskUpdateHandler = (state: ComputeState, event: any) => {
  const task = extractTaskFromEvent(event);
  if (task) {
    const newTasks = new Map(state.tasks);
    const existingTask = newTasks.get(task.id);

    // Perform a shallow comparison for tasks as well
    if (
      existingTask &&
      existingTask.status === task.status &&
      existingTask.progress === task.progress &&
      existingTask.result === task.result &&
      existingTask.error === task.error &&
      existingTask.requirements === task.requirements &&
      // Shallow array comparison for assignedWorkerIds
      existingTask.assignedWorkerIds?.length === task.assignedWorkerIds?.length &&
      existingTask.assignedWorkerIds?.every((id, i) => id === task.assignedWorkerIds?.[i])
    ) {
      return state; // No change, return current state
    }

    if (existingTask) {
      newTasks.set(task.id, { ...existingTask, ...task });
    } else {
      newTasks.set(task.id, task);
    }
    return { tasks: newTasks };
  }
  return state;
};

const metricsUpdateHandler = (state: ComputeState, event: any) => {
  const payload = getEventPayload(event);
  if (payload) {
    const currentMetrics = state.metrics;
    // Shallow compare payload with current metrics to avoid unnecessary updates
    if (
      currentMetrics &&
      Object.keys(payload).every(key => (payload as any)[key] === (currentMetrics as any)[key])
    ) {
      return state; // No change, return current state
    }

    return {
      metrics: payload
    };
  }
  return state;
};

export const useComputeStore = create<ComputeStore>()(
  devtools(
    set => ({
      ...initialState,

      setUserId: userId => set({ userId }, false, 'setUserId'),
      setComputeEnabled: isEnabled =>
        set({ isComputeEnabled: isEnabled }, false, 'setComputeEnabled'),

      reset: () => set(initialState, false, 'resetComputeStore'),

      handleWorkerUpdate: event => {
        set(state => workerUpdateHandler(state, event), false, 'handleWorkerUpdate');
      },

      handleTaskUpdate: event => {
        set(state => taskUpdateHandler(state, event), false, 'handleTaskUpdate');
      },

      handleMetricsUpdate: event => {
        set(state => metricsUpdateHandler(state, event), false, 'handleMetricsUpdate');
      }
    }),
    {
      name: 'compute-store'
    }
  )
);

// --- Subscribe to centralized event store ---
(() => {
  const { subscribe } = useEventStore.getState();

  // Subscribe to all events and filter for compute-related ones.
  // This is more efficient than multiple subscriptions if we have many event types.
  subscribe('*', event => {
    const { type } = event;
    // Get fresh handlers from the store in case of HMR
    const { handleTaskUpdate, handleWorkerUpdate, handleMetricsUpdate } =
      useComputeStore.getState();

    if (type.startsWith('compute:task') || type.startsWith('compute:dispatch')) {
      handleTaskUpdate(event);
    } else if (type.startsWith('compute:capabilities')) {
      handleWorkerUpdate(event);
    } else if (type.startsWith('compute:metrics')) {
      handleMetricsUpdate(event);
    }
  });
})();

// --- Selectors ---

export const selectWorkers = (state: ComputeState): ComputeWorker[] =>
  Array.from(state.workers.values());

export const selectTasks = (state: ComputeState): ComputeTask[] => Array.from(state.tasks.values());

export const selectDerivedSystemMetrics = (state: ComputeState) => {
  const workers = selectWorkers(state);
  const tasks = selectTasks(state);

  const totalWorkers = workers.length;
  const activeWorkers = workers.filter(w => w.status === 'active').length;
  const totalLoad = workers.reduce((acc, w) => acc + (w.currentLoad || 0), 0);
  const avgLoad = totalWorkers > 0 ? (totalLoad / totalWorkers) * 100 : 0;

  const totalTasks = tasks.length;
  const activeTasks = tasks.filter(
    t => t.status === 'in-progress' || t.status === 'assigned' || t.status === 'pending'
  ).length;
  const completedTasks = tasks.filter(t => t.status === 'completed').length;
  const failedTasks = tasks.filter(t => t.status === 'failed').length;

  return {
    totalWorkers,
    activeWorkers,
    avgLoad: Math.round(avgLoad),
    totalTasks,
    activeTasks,
    completedTasks,
    failedTasks
  };
};
