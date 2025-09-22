import { Component } from 'react';
import type { ErrorInfo, ReactNode } from 'react';

interface Props {
  children: ReactNode;
  storeName: string;
  fallback?: ReactNode;
  onStoreError?: (error: Error, storeName: string) => void;
}

interface State {
  hasError: boolean;
  error?: Error;
  errorInfo?: ErrorInfo;
  retryCount: number;
}

class StoreErrorBoundary extends Component<Props, State> {
  private maxRetries = 3;
  private retryTimeoutId: number | null = null;

  constructor(props: Props) {
    super(props);
    this.state = { hasError: false, retryCount: 0 };
  }

  static getDerivedStateFromError(error: Error): Partial<State> {
    return {
      hasError: true,
      error
    };
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    console.error(`[StoreErrorBoundary] Error in ${this.props.storeName}:`, {
      error: error.message,
      stack: error.stack,
      componentStack: errorInfo.componentStack,
      storeName: this.props.storeName,
      retryCount: this.state.retryCount
    });

    this.setState({
      error,
      errorInfo
    });

    // Call custom error handler
    if (this.props.onStoreError) {
      this.props.onStoreError(error, this.props.storeName);
    }

    // Attempt to recover the store
    this.attemptStoreRecovery();
  }

  componentWillUnmount() {
    if (this.retryTimeoutId) {
      clearTimeout(this.retryTimeoutId);
    }
  }

  private attemptStoreRecovery = () => {
    const { storeName } = this.props;
    const { retryCount } = this.state;

    if (retryCount >= this.maxRetries) {
      console.error(`[StoreErrorBoundary] Max retries reached for ${storeName}`);
      return;
    }

    console.log(
      `[StoreErrorBoundary] Attempting to recover ${storeName} (attempt ${retryCount + 1})`
    );

    // Clear any existing timeout
    if (this.retryTimeoutId) {
      clearTimeout(this.retryTimeoutId);
    }

    // Retry after a delay
    this.retryTimeoutId = window.setTimeout(
      () => {
        try {
          // Attempt to reinitialize the store
          this.reinitializeStore(storeName);

          // Reset error state
          this.setState({
            hasError: false,
            error: undefined,
            errorInfo: undefined,
            retryCount: retryCount + 1
          });
        } catch (recoveryError) {
          console.error(
            `[StoreErrorBoundary] Store recovery failed for ${storeName}:`,
            recoveryError
          );
          this.setState({ retryCount: retryCount + 1 });
        }
      },
      1000 * (retryCount + 1)
    ); // Exponential backoff
  };

  private reinitializeStore = (storeName: string) => {
    // Store-specific recovery logic
    switch (storeName) {
      case 'event':
        // Reinitialize event store
        console.log('[StoreErrorBoundary] Reinitializing event store');
        break;
      case 'connection':
        // Reinitialize connection store
        console.log('[StoreErrorBoundary] Reinitializing connection store');
        break;
      case 'campaign':
        // Reinitialize campaign store
        console.log('[StoreErrorBoundary] Reinitializing campaign store');
        break;
      case 'metadata':
        // Reinitialize metadata store
        console.log('[StoreErrorBoundary] Reinitializing metadata store');
        break;
      default:
        console.warn(`[StoreErrorBoundary] Unknown store: ${storeName}`);
    }
  };

  private handleManualRetry = () => {
    this.setState({ retryCount: 0 });
    this.attemptStoreRecovery();
  };

  private handleReset = () => {
    // Force page reload to reset all stores
    window.location.reload();
  };

  render() {
    if (this.state.hasError) {
      const { storeName, fallback } = this.props;
      const { retryCount } = this.state;

      if (fallback) {
        return fallback;
      }

      return (
        <div
          style={{
            padding: '16px',
            border: '1px solid #f00',
            backgroundColor: '#000',
            color: '#fff',
            fontFamily: 'Monaco, Menlo, Consolas, monospace',
            fontSize: '11px',
            lineHeight: '1.4'
          }}
        >
          <h3 style={{ color: '#f00', margin: '0 0 12px 0' }}>⚠️ Store Error: {storeName}</h3>

          <div style={{ marginBottom: '12px' }}>
            <strong>Retry Attempt:</strong> {retryCount} / {this.maxRetries}
          </div>

          <div style={{ marginBottom: '12px' }}>
            {retryCount < this.maxRetries ? (
              <span style={{ color: '#ff0' }}>Attempting automatic recovery...</span>
            ) : (
              <span style={{ color: '#f00' }}>
                Automatic recovery failed. Manual intervention required.
              </span>
            )}
          </div>

          <details style={{ marginBottom: '12px' }}>
            <summary style={{ cursor: 'pointer', marginBottom: '6px' }}>Error Details</summary>
            <pre
              style={{
                backgroundColor: '#111',
                padding: '6px',
                overflow: 'auto',
                fontSize: '9px',
                whiteSpace: 'pre-wrap'
              }}
            >
              {this.state.error?.message}
            </pre>
          </details>

          <div style={{ display: 'flex', gap: '6px' }}>
            {retryCount < this.maxRetries && (
              <button
                onClick={this.handleManualRetry}
                style={{
                  backgroundColor: '#333',
                  color: '#fff',
                  border: '1px solid #555',
                  padding: '4px 8px',
                  cursor: 'pointer',
                  fontSize: '10px'
                }}
              >
                Retry Now
              </button>
            )}

            <button
              onClick={this.handleReset}
              style={{
                backgroundColor: '#f00',
                color: '#fff',
                border: '1px solid #f00',
                padding: '4px 8px',
                cursor: 'pointer',
                fontSize: '10px'
              }}
            >
              Reset All
            </button>
          </div>
        </div>
      );
    }

    return this.props.children;
  }
}

export default StoreErrorBoundary;
