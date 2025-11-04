
export interface ComputeWorker {
  id: string;
  cpuCores: number;
  memoryMb: number;
  wasm: boolean;
  threads: boolean;
  simd: boolean;
  webgpu: boolean;
  gpuBackend?: string;
  gpuFeatures?: string[];
  currentLoad?: number;
  status: 'active' | 'inactive' | 'overloaded';
}

export interface ComputeTask {
  id: string;
  status: 'pending' | 'assigned' | 'in-progress' | 'completed' | 'failed' | 'cancelled';
  requirements: any;
  assignedWorkerIds?: string[];
  progress?: number;
  result?: any;
  error?: string;
}

export interface ComputeMetrics {
  activeWorkers: number;
  totalWorkers: number;
  totalTasks: number;
  activeTasks: number;
  completedTasks: number;
  failedTasks: number;
  avgLoad: number;
  connectionStatus: 'connected' | 'connecting' | 'disconnected';
  cpuCoresTotal?: number;
  averageCpuCores?: number;
  totalMemoryMb?: number;
  averageMemoryMb?: number;
}
