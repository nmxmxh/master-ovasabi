import { Component } from 'react';
import type { ReactNode, ErrorInfo } from 'react';
import styled from 'styled-components';

interface Props {
  children: ReactNode;
  fallback?: ReactNode;
}

interface State {
  hasError: boolean;
  error: Error | null;
  errorInfo: ErrorInfo | null;
}

const ErrorContainer = styled.div`
  background: rgba(239, 68, 68, 0.1);
  border: 1px solid rgba(239, 68, 68, 0.3);
  border-radius: 8px;
  padding: 16px;
  margin: 16px 0;
`;

const ErrorTitle = styled.h3`
  color: #dc2626;
  margin: 0 0 12px 0;
  font-size: 16px;
  font-weight: 600;
`;

const ErrorMessage = styled.p`
  color: #7f1d1d;
  margin: 0 0 12px 0;
  font-size: 14px;
`;

const ErrorDetails = styled.details`
  color: #7f1d1d;
  font-size: 12px;
  margin-top: 8px;

  summary {
    cursor: pointer;
    font-weight: 500;
    margin-bottom: 8px;
  }

  pre {
    background: rgba(0, 0, 0, 0.05);
    padding: 8px;
    border-radius: 4px;
    overflow: auto;
    white-space: pre-wrap;
  }
`;

const RetryButton = styled.button`
  background: #dc2626;
  color: white;
  border: none;
  padding: 8px 16px;
  border-radius: 4px;
  cursor: pointer;
  font-size: 14px;
  font-weight: 500;
  transition: background-color 0.2s;

  &:hover {
    background: #b91c1c;
  }
`;

class ErrorBoundary extends Component<Props, State> {
  constructor(props: Props) {
    super(props);
    this.state = {
      hasError: false,
      error: null,
      errorInfo: null
    };
  }

  static getDerivedStateFromError(error: Error): State {
    return {
      hasError: true,
      error,
      errorInfo: null
    };
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    console.error('[ErrorBoundary] Component error caught:', error, errorInfo);
    this.setState({
      error,
      errorInfo
    });

    // Log to external service if needed
    if (typeof window !== 'undefined' && (window as any).logStatus) {
      (window as any).logStatus('React Error Boundary', {
        error: error.message,
        stack: error.stack,
        errorInfo
      });
    }
  }

  handleRetry = () => {
    this.setState({
      hasError: false,
      error: null,
      errorInfo: null
    });
  };

  render() {
    if (this.state.hasError) {
      if (this.props.fallback) {
        return this.props.fallback;
      }

      return (
        <ErrorContainer>
          <ErrorTitle>⚠️ Component Error</ErrorTitle>
          <ErrorMessage>
            {this.state.error?.message || 'An unexpected error occurred in this component.'}
          </ErrorMessage>

          <RetryButton onClick={this.handleRetry}>Retry Component</RetryButton>

          {this.state.error && (
            <ErrorDetails>
              <summary>Technical Details</summary>
              <pre>
                <strong>Error:</strong> {this.state.error.name}: {this.state.error.message}
                {this.state.error.stack && (
                  <>
                    <br />
                    <br />
                    <strong>Stack Trace:</strong>
                    <br />
                    {this.state.error.stack}
                  </>
                )}
                {this.state.errorInfo && (
                  <>
                    <br />
                    <br />
                    <strong>Component Stack:</strong>
                    <br />
                    {this.state.errorInfo.componentStack}
                  </>
                )}
              </pre>
            </ErrorDetails>
          )}
        </ErrorContainer>
      );
    }

    return this.props.children;
  }
}

export default ErrorBoundary;
