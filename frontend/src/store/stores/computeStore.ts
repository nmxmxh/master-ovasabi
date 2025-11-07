import { create } from 'zustand';
import { devtools } from 'zustand/middleware';
import type { ComputeWorker, ComputeTask, ComputeMetrics } from '../types/compute';

// --- Helper functions to extract data from events ---

const extractWorkerFromEvent = (event: any): ComputeWorker | null => {
  try {
    console.log('extractWorkerFromEvent: Incoming event', event);
    const source =
      event.metadata?.global_context?.source || event.metadata?.global_context?.device_id;
    if (!source) {
      console.log('extractWorkerFromEvent: No source found, returning null');
      return null;
    }

    const payload = event.payload?.data || event.payload;
    const caps = payload || {};

    return {
      id: source,
      cpuCores: caps.cpu_cores || 0,
      memoryMb: caps.memory_mb || 0,
      wasm: caps.wasm || false,
      threads: caps.threads || false,
      simd: caps.simd || false,
      webgpu: caps.webgpu || false,
      gpuBackend: caps.gpu?.backend || undefined,
      gpuFeatures: caps.gpu?.features || [],
      status: 'active' as const,
      currentLoad: 0
    };
  } catch (error) {
    console.error('Failed to extract worker from event:', error);
    return null;
  }
};

const extractTaskFromEvent = (event: any): ComputeTask | null => {
  try {
    // For CustomEvent, the data is in event.detail
    const payload = event.detail || event.payload?.data || event.payload;
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
  const payload = event.payload?.data || event.payload;
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

// --- Initialize WASM Listeners once ---
(() => {
  const { handleWorkerUpdate, handleTaskUpdate } = useComputeStore.getState();

  // Listen for specific custom events dispatched from WASM
  window.addEventListener('compute:task', handleTaskUpdate);
  window.addEventListener('compute:capabilities:v1:update', handleWorkerUpdate);

  // Assign a general message handler for other events coming through onWasmMessage
  (window as any).onWasmMessage = (event: any) => {
    if (!event || !event.type) {
      return;
    }

    const { type } = event;
    // Get fresh handlers, in case they are updated (e.g. HMR)
    const { handleTaskUpdate, handleWorkerUpdate, handleMetricsUpdate } =
      useComputeStore.getState();

    if (type.startsWith('compute:task')) {
      handleTaskUpdate(event);
    } else if (type.startsWith('compute:dispatch')) {
      handleTaskUpdate(event);
    } else if (type.startsWith('compute:capabilities')) {
      handleWorkerUpdate(event);
    } else if (type.startsWith('compute:metrics')) {
      handleMetricsUpdate(event);
    }
  };
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
