import React, { useState, useMemo } from 'react';
import { useEmitEvent } from '../store/hooks/useEvents';
import { useConnectionStore } from '../store/stores/connectionStore';
import {
  useComputeStore,
  selectWorkers,
  selectTasks,
  selectDerivedSystemMetrics
} from '../store/stores/computeStore';

// Re-using minimal styles from App.tsx for consistency
const minimalStyles = `
  .minimal-app {
    font-family: 'Monaco', 'Menlo', 'Consolas', monospace;
    background: #000;
    color: #fff;
    font-size: 12px;
    line-height: 1.4;
    min-height: 100vh;
    display: flex;
    flex-direction: column;
  }
  
  .minimal-header {
    border-bottom: 1px solid #333;
    padding: 8px 16px;
    background: #111;
    flex-shrink: 0;
  }
  
  .minimal-nav {
    display: flex;
    gap: 16px;
  }
  
  .minimal-link {
    color: #fff;
    text-decoration: none;
    padding: 4px 8px;
    border: 1px solid #333;
    background: #000;
    font-size: 11px;
    cursor: pointer;
    display: inline-block;
    transition: all 0.2s ease;
  }
  
  .minimal-link:hover {
    background: #333;
    transform: translateY(-1px);
  }
  
  .minimal-link.active {
    background: #fff;
    color: #000;
  }
  
  .minimal-main {
    padding: 16px;
    max-width: 1200px;
    margin: 0 auto;
    flex: 1;
    width: 100%;
  }
  
  .minimal-section {
    margin-bottom: 24px;
    border: 1px solid #333;
    padding: 12px;
    background: #111;
  }
  
  .minimal-title {
    font-size: 14px;
    font-weight: bold;
    margin-bottom: 8px;
    color: #fff;
  }
  
  .minimal-text {
    font-size: 11px;
    color: #ccc;
    margin-bottom: 4px;
  }
  
  .minimal-button {
    background: #000;
    color: #fff;
    border: 1px solid #333;
    padding: 4px 8px;
    font-size: 11px;
    cursor: pointer;
    font-family: inherit;
    transition: all 0.2s ease;
  }
  
  .minimal-button:hover {
    background: #333;
    transform: translateY(-1px);
  }
  
  .minimal-button:disabled {
    opacity: 0.5;
    cursor: not-allowed;
    transform: none;
  }
  
  .minimal-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
    gap: 16px;
  }
  
  .minimal-card {
    border: 1px solid #333;
    padding: 16px;
    background: #000;
    display: flex;
    flex-direction: column;
    min-height: 200px;
    transition: all 0.2s ease;
    position: relative;
  }
  
  .minimal-card:hover {
    border-color: #555;
    transform: translateY(-2px);
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
  }
  
  .minimal-card.active {
    border: 2px solid #fff;
    box-shadow: 0 0 8px rgba(255, 255, 255, 0.2);
  }
  
  .minimal-card.active:hover {
    border-color: #fff;
    box-shadow: 0 0 12px rgba(255, 255, 255, 0.3);
  }
  
  .minimal-card-header {
    margin-bottom: 12px;
    flex-shrink: 0;
  }
  
  .minimal-card-title {
    font-size: 13px;
    font-weight: bold;
    color: #fff;
    margin-bottom: 6px;
    line-height: 1.3;
    display: -webkit-box;
    -webkit-line-clamp: 2;
    -webkit-box-orient: vertical;
    overflow: hidden;
  }
  
  .minimal-card-meta {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 8px;
    flex-wrap: wrap;
  }
  
  .minimal-card-id {
    font-size: 10px;
    color: #888;
    font-family: 'Monaco', 'Menlo', 'Consolas', monospace;
  }
  
  .minimal-card-description {
    font-size: 11px;
    color: #ccc;
    line-height: 1.4;
    margin-bottom: 12px;
    flex: 1;
    display: -webkit-box;
    -webkit-line-clamp: 3;
    -webkit-box-orient: vertical;
    overflow: hidden;
  }
  
  .minimal-card-features {
    font-size: 10px;
    color: #999;
    margin-bottom: 12px;
    display: -webkit-box;
    -webkit-line-clamp: 2;
    -webkit-box-orient: vertical;
    overflow: hidden;
  }
  
  .minimal-card-actions {
    margin-top: auto;
    padding-top: 12px;
    border-top: 1px solid #222;
  }
  
  .minimal-code {
    background: #000;
    color: #0f0;
    padding: 8px;
    border: 1px solid #333;
    font-family: 'Monaco', 'Menlo', 'Consolas', monospace;
    font-size: 10px;
    overflow-x: auto;
    white-space: pre-wrap;
  }
  
  .minimal-status {
    display: inline-block;
    padding: 2px 6px;
    font-size: 10px;
    border: 1px solid #333;
    border-radius: 2px;
    font-weight: bold;
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }
  
  .minimal-status.active {
    background: #0f0;
    color: #000;
    border-color: #0f0;
  }
  
  .minimal-status.inactive {
    background: #f00;
    color: #fff;
    border-color: #f00;
  }
  
  .minimal-status.loading {
    background: #ff0;
    color: #000;
    border-color: #ff0;
  }
  
  .minimal-card-button {
    width: 100%;
    background: #000;
    color: #fff;
    border: 1px solid #333;
    padding: 8px 12px;
    font-size: 11px;
    cursor: pointer;
    font-family: inherit;
    transition: all 0.2s ease;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    font-weight: bold;
  }
  
  .minimal-card-button:hover {
    background: #333;
    border-color: #555;
    transform: translateY(-1px);
  }
  
  .minimal-card-button:active {
    transform: translateY(0);
  }
`;

// --- ComputeDashboardPage Component ---

const ComputeDashboardPage: React.FC = () => {
  const emitEvent = useEmitEvent();
  const { connected, wasmReady } = useConnectionStore();

  // --- New state management from computeStore ---
  const workers = useComputeStore(selectWorkers);
  const tasks = useComputeStore(selectTasks);
  const derivedMetrics = useComputeStore(selectDerivedSystemMetrics);
  const backendMetrics = useComputeStore(state => state.metrics);

  const [newTaskRequirements, setNewTaskRequirements] = useState<string>('');
  const [submissionStatus, setSubmissionStatus] = useState<
    'idle' | 'submitting' | 'success' | 'error'
  >('idle');
  const [submissionError, setSubmissionError] = useState<string | null>(null);

  // System health metrics
  const systemMetrics = useMemo(() => {
    const connectionStatus =
      connected && wasmReady ? 'connected' : connected ? 'connecting' : 'disconnected';

    // Combine client-derived metrics with backend-provided metrics
    return {
      ...derivedMetrics,
      totalWorkers: backendMetrics?.totalWorkers ?? derivedMetrics.totalWorkers,
      activeWorkers: backendMetrics?.activeWorkers ?? derivedMetrics.activeWorkers,
      cpuCoresTotal: backendMetrics?.cpuCoresTotal,
      averageCpuCores: backendMetrics?.averageCpuCores,
      totalMemoryMb: backendMetrics?.totalMemoryMb,
      averageMemoryMb: backendMetrics?.averageMemoryMb,
      connectionStatus
    };
  }, [derivedMetrics, backendMetrics, connected, wasmReady]);

  const handleTaskSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newTaskRequirements.trim()) return;

    setSubmissionStatus('submitting');
    setSubmissionError(null);
    try {
      let requirements: any;
      try {
        requirements = JSON.parse(newTaskRequirements);
      } catch (err: any) {
        throw new Error(err?.message || 'Invalid JSON');
      }

      // Emit compute dispatch requested event
      emitEvent({
        type: 'compute:dispatch:v1:requested',
        payload: {
          data: {
            task_id: `task-${Date.now()}-${Math.random().toString(36).substring(7)}`,
            requirements: requirements,
            inputs: requirements.input_data_uri
              ? [
                  {
                    uri: requirements.input_data_uri
                  }
                ]
              : [],
            module: requirements.module || undefined,
            params: requirements.params || {}
          }
        },
        metadata: {
          global_context: {}
        }
      });

      setNewTaskRequirements('');
      setSubmissionStatus('success');
      setTimeout(() => setSubmissionStatus('idle'), 3000);
    } catch (error: any) {
      console.error('Failed to submit task:', error);
      setSubmissionError(error.message || 'Invalid JSON or API error');
      setSubmissionStatus('error');
    }
  };

  const getStatusClass = (status: string) => {
    switch (status) {
      case 'active':
      case 'completed':
        return 'active';
      case 'inactive':
      case 'failed':
      case 'overloaded':
        return 'inactive';
      case 'pending':
      case 'in-progress':
      case 'assigned':
        return 'loading';
      default:
        return '';
    }
  };

  return (
    <div className="minimal-main">
      <style>{minimalStyles}</style>

      {/* System Status Section */}
      <div className="minimal-section">
        <div className="minimal-title">SYSTEM STATUS</div>
        <div
          className="minimal-grid"
          style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))', marginTop: '12px' }}
        >
          <div className="minimal-card">
            <div className="minimal-card-title">Connection</div>
            <div className="minimal-card-meta">
              <span
                className={`minimal-status ${systemMetrics.connectionStatus === 'connected' ? 'active' : systemMetrics.connectionStatus === 'connecting' ? 'loading' : 'inactive'}`}
              >
                {systemMetrics.connectionStatus.toUpperCase()}
              </span>
            </div>
            <div className="minimal-card-description">
              WASM: {wasmReady ? 'READY' : 'NOT READY'}
              <br />
              WebSocket: {connected ? 'CONNECTED' : 'DISCONNECTED'}
            </div>
          </div>
          <div className="minimal-card">
            <div className="minimal-card-title">Workers</div>
            <div className="minimal-card-description">
              Active: {systemMetrics.activeWorkers}
              <br />
              Total (backend): {systemMetrics.totalWorkers}
              <br />
              Avg Load: {systemMetrics.avgLoad}%
            </div>
          </div>
          <div className="minimal-card">
            <div className="minimal-card-title">Tasks</div>
            <div className="minimal-card-description">
              Total: {systemMetrics.totalTasks}
              <br />
              Active: {systemMetrics.activeTasks}
              <br />
              Completed: {systemMetrics.completedTasks}
              <br />
              Failed: {systemMetrics.failedTasks}
            </div>
          </div>
          <div className="minimal-card">
            <div className="minimal-card-title">Resources</div>
            <div className="minimal-card-description">
              CPU Total (backend): {systemMetrics.cpuCoresTotal ?? '—'}
              <br />
              CPU Avg (backend): {systemMetrics.averageCpuCores ?? '—'}
              <br />
              Mem Total MB (backend): {systemMetrics.totalMemoryMb ?? '—'}
              <br />
              Mem Avg MB (backend): {systemMetrics.averageMemoryMb ?? '—'}
            </div>
          </div>
        </div>
      </div>

      {/* Task Submission */}
      <div className="minimal-section">
        <div className="minimal-title">SUBMIT NEW TASK</div>
        <form onSubmit={handleTaskSubmit}>
          <textarea
            className="minimal-code"
            rows={8}
            placeholder={`Enter task requirements (JSON format):
{
  "min": {
    "wasm": true,
    "cpuCores": 2
  },
  "preferred": {
    "webgpu": true
  },
  "parallelism_strategy": "map",
  "input_data_uri": "s3://my-bucket/large-file.json"
}`}
            value={newTaskRequirements}
            onChange={e => setNewTaskRequirements(e.target.value)}
            style={{ width: '100%', marginBottom: '8px' }}
          ></textarea>
          <button
            type="submit"
            className="minimal-button"
            disabled={submissionStatus === 'submitting'}
          >
            {submissionStatus === 'submitting' ? 'SUBMITTING...' : 'SUBMIT TASK'}
          </button>
          {submissionStatus === 'success' && (
            <span style={{ color: '#0f0', marginLeft: '8px' }}>Task submitted successfully!</span>
          )}
          {submissionStatus === 'error' && (
            <span style={{ color: '#f00', marginLeft: '8px' }}>Error: {submissionError}</span>
          )}
        </form>
      </div>

      {/* Workers List */}
      <div className="minimal-section">
        <div className="minimal-title">
          ACTIVE WORKERS ({workers.length})
          {!connected && (
            <span style={{ color: '#ff0', marginLeft: '8px', fontSize: '10px' }}>
              ⚠️ NOT CONNECTED
            </span>
          )}
        </div>
        {!connected || !wasmReady ? (
          <div className="minimal-text" style={{ color: '#ff0' }}>
            ⚠️ WebSocket connection required to view workers. Waiting for connection...
          </div>
        ) : workers.length === 0 ? (
          <div className="minimal-text">
            No active workers found. Workers will appear here when they announce their capabilities
            via compute:capabilities:v1:update events.
          </div>
        ) : (
          <div className="minimal-grid">
            {workers.map(worker => (
              <div
                key={worker.id}
                className={`minimal-card ${worker.status === 'active' ? 'active' : ''}`}
              >
                <div className="minimal-card-header">
                  <div className="minimal-card-title">Worker ID: {worker.id}</div>
                  <div className="minimal-card-meta">
                    <span className={`minimal-status ${getStatusClass(worker.status)}`}>
                      {worker.status.toUpperCase()}
                    </span>
                    {worker.currentLoad !== undefined && (
                      <span className="minimal-card-id">
                        Load: {(worker.currentLoad * 100).toFixed(0)}%
                      </span>
                    )}
                  </div>
                </div>
                <div className="minimal-card-description">
                  CPU: {worker.cpuCores} cores, Memory: {worker.memoryMb}MB
                  <br />
                  WASM: {worker.wasm ? 'Yes' : 'No'}, Threads: {worker.threads ? 'Yes' : 'No'}
                  <br />
                  SIMD: {worker.simd ? 'Yes' : 'No'}, WebGPU: {worker.webgpu ? 'Yes' : 'No'}
                  {worker.gpuBackend && (
                    <>
                      <br />
                      GPU: {worker.gpuBackend} ({worker.gpuFeatures?.join(', ')})
                    </>
                  )}
                </div>
                <div className="minimal-card-actions">
                  {/* Add worker-specific actions here */}
                  <button className="minimal-card-button" disabled>
                    View Details
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Tasks List */}
      <div className="minimal-section">
        <div className="minimal-title">
          COMPUTE TASKS ({tasks.length})
          {tasks.length > 0 && (
            <span style={{ color: '#0f0', marginLeft: '8px', fontSize: '10px' }}>● LIVE</span>
          )}
        </div>
        {tasks.length === 0 ? (
          <div className="minimal-text">
            No compute tasks found. Tasks will appear here when compute:dispatch:v1:* events are
            received.
          </div>
        ) : (
          <div className="minimal-grid">
            {tasks.map(task => (
              <div key={task.id} className="minimal-card">
                <div className="minimal-card-header">
                  <div className="minimal-card-title">Task ID: {task.id}</div>
                  <div className="minimal-card-meta">
                    <span className={`minimal-status ${getStatusClass(task.status)}`}>
                      {task.status.toUpperCase()}
                    </span>
                    {task.progress !== undefined && (
                      <span className="minimal-card-id">Progress: {task.progress}%</span>
                    )}
                  </div>
                </div>
                <div className="minimal-card-description">
                  Requirements:{' '}
                  <pre className="minimal-code">{JSON.stringify(task.requirements, null, 2)}</pre>
                  {task.assignedWorkerIds && task.assignedWorkerIds.length > 0 && (
                    <div className="minimal-text">
                      Assigned to: {task.assignedWorkerIds.join(', ')}
                    </div>
                  )}
                  {task.error && (
                    <div className="minimal-text" style={{ color: '#f00' }}>
                      Error:{' '}
                      {typeof task.error === 'object' ? JSON.stringify(task.error) : task.error}
                    </div>
                  )}
                </div>
                <div className="minimal-card-actions">
                  {/* Add task-specific actions here */}
                  <button className="minimal-card-button" disabled>
                    View Logs
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Network Visualization */}
      {workers.length > 0 && (
        <div className="minimal-section">
          <div className="minimal-title">NETWORK VISION</div>
          <div
            style={{
              position: 'relative',
              width: '100%',
              height: '400px',
              border: '1px solid #333',
              background: '#000',
              overflow: 'hidden'
            }}
          >
            <svg width="100%" height="100%" style={{ position: 'absolute', top: 0, left: 0 }}>
              {/* Draw connections between workers and tasks */}
              {tasks.map((task, taskIndex) => {
                if (!task.assignedWorkerIds || task.assignedWorkerIds.length === 0) return null;
                return task.assignedWorkerIds.map(workerId => {
                  const worker = workers.find(w => w.id === workerId);
                  if (!worker) return null;

                  const workerIdx = workers.findIndex(w => w.id === workerId);

                  // Calculate positions
                  const workerX = 100 + (workerIdx % 4) * 200;
                  const workerY = 50 + Math.floor(workerIdx / 4) * 150;
                  const taskX = 100 + (taskIndex % 4) * 200;
                  const taskY = 250 + Math.floor(taskIndex / 4) * 150;

                  return (
                    <line
                      key={`${task.id}-${workerId}`}
                      x1={workerX}
                      y1={workerY}
                      x2={taskX}
                      y2={taskY}
                      stroke={
                        task.status === 'completed'
                          ? '#0f0'
                          : task.status === 'failed'
                            ? '#f00'
                            : '#0ff'
                      }
                      strokeWidth="1"
                      opacity="0.3"
                    />
                  );
                });
              })}

              {/* Draw workers as nodes */}
              {workers.map((worker, idx) => {
                const x = 100 + (idx % 4) * 200;
                const y = 50 + Math.floor(idx / 4) * 150;
                const loadColor =
                  worker.currentLoad && worker.currentLoad > 0.8
                    ? '#ff0'
                    : worker.currentLoad && worker.currentLoad > 0.5
                      ? '#0ff'
                      : '#0f0';

                return (
                  <g key={worker.id}>
                    <circle
                      cx={x}
                      cy={y}
                      r={20}
                      fill={worker.status === 'active' ? loadColor : '#666'}
                      stroke="#fff"
                      strokeWidth="2"
                    />
                    <text
                      x={x}
                      y={y + 40}
                      fill="#fff"
                      fontSize="10"
                      textAnchor="middle"
                      style={{ fontFamily: 'Monaco, Menlo, Consolas, monospace' }}
                    >
                      {worker.id.substring(0, 12)}
                    </text>
                    {worker.webgpu && (
                      <text
                        x={x}
                        y={y}
                        fill="#000"
                        fontSize="8"
                        textAnchor="middle"
                        fontWeight="bold"
                      >
                        GPU
                      </text>
                    )}
                  </g>
                );
              })}

              {/* Draw tasks as nodes */}
              {tasks.map((task, idx) => {
                const x = 100 + (idx % 4) * 200;
                const y = 250 + Math.floor(idx / 4) * 150;
                const statusColor =
                  task.status === 'completed'
                    ? '#0f0'
                    : task.status === 'failed'
                      ? '#f00'
                      : task.status === 'in-progress'
                        ? '#0ff'
                        : '#ff0';

                return (
                  <g key={task.id}>
                    <rect
                      x={x - 25}
                      y={y - 15}
                      width="50"
                      height="30"
                      fill={statusColor}
                      stroke="#fff"
                      strokeWidth="1"
                      opacity="0.7"
                    />
                    <text
                      x={x}
                      y={y + 30}
                      fill="#fff"
                      fontSize="9"
                      textAnchor="middle"
                      style={{ fontFamily: 'Monaco, Menlo, Consolas, monospace' }}
                    >
                      {task.id.substring(0, 12)}
                    </text>
                    <text
                      x={x}
                      y={y + 3}
                      fill="#000"
                      fontSize="8"
                      textAnchor="middle"
                      fontWeight="bold"
                    >
                      {task.status.substring(0, 4).toUpperCase()}
                    </text>
                  </g>
                );
              })}
            </svg>

            {/* Legend */}
            <div
              style={{
                position: 'absolute',
                bottom: '10px',
                right: '10px',
                background: '#111',
                border: '1px solid #333',
                padding: '8px',
                fontSize: '10px',
                fontFamily: 'Monaco, Menlo, Consolas, monospace'
              }}
            >
              <div style={{ marginBottom: '4px' }}>
                <span style={{ color: '#0f0' }}>●</span> Workers (Active)
              </div>
              <div style={{ marginBottom: '4px' }}>
                <span style={{ color: '#0ff' }}>●</span> Tasks (In Progress)
              </div>
              <div style={{ marginBottom: '4px' }}>
                <span style={{ color: '#0f0' }}>●</span> Tasks (Completed)
              </div>
              <div>
                <span style={{ color: '#f00' }}>●</span> Tasks (Failed)
              </div>
            </div>
          </div>
          <div
            className="minimal-text"
            style={{ marginTop: '8px', fontSize: '10px', color: '#888' }}
          >
            Network visualization showing connections between workers (circles) and tasks
            (rectangles). Lines represent task assignments. Colors indicate status and load.
          </div>
        </div>
      )}
    </div>
  );
};

export default ComputeDashboardPage;
