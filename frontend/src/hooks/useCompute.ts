import { useMemo } from 'react';
import {
  useComputeStore,
  selectWorkers,
  selectTasks,
  selectDerivedSystemMetrics
} from '../store/stores/computeStore';

export const useCompute = () => {
  const workers = useComputeStore(selectWorkers);
  const tasks = useComputeStore(selectTasks);
  const derivedMetrics = useComputeStore(selectDerivedSystemMetrics);
  const backendMetrics = useComputeStore(state => state.metrics);
  const isComputeEnabled = useComputeStore(state => state.isComputeEnabled);
  const userId = useComputeStore(state => state.userId);
  const setUserId = useComputeStore(state => state.setUserId);
  const setComputeEnabled = useComputeStore(state => state.setComputeEnabled);
  const reset = useComputeStore(state => state.reset);

  const metrics = useMemo(
    () => ({
      ...derivedMetrics,
      totalWorkers: backendMetrics?.totalWorkers ?? derivedMetrics.totalWorkers,
      activeWorkers: backendMetrics?.activeWorkers ?? derivedMetrics.activeWorkers,
      cpuCoresTotal: backendMetrics?.cpuCoresTotal ?? undefined,
      averageCpuCores: backendMetrics?.averageCpuCores ?? undefined,
      totalMemoryMb: backendMetrics?.totalMemoryMb ?? undefined,
      averageMemoryMb: backendMetrics?.averageMemoryMb ?? undefined
    }),
    [backendMetrics, derivedMetrics]
  );

  return {
    userId,
    workers,
    tasks,
    metrics,
    isComputeEnabled,
    setUserId,
    setComputeEnabled,
    reset
  };
};
