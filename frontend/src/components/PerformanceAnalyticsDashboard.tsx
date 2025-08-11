import React, { useState, useEffect, useCallback } from 'react';
import styled, { keyframes, css } from 'styled-components';

// Animations
const pulseGlow = keyframes`
  0% { box-shadow: 0 0 5px rgba(0, 123, 255, 0.5); }
  50% { box-shadow: 0 0 20px rgba(0, 123, 255, 0.8); }
  100% { box-shadow: 0 0 5px rgba(0, 123, 255, 0.5); }
`;

const slideInFromRight = keyframes`
  from { transform: translateX(100%); opacity: 0; }
  to { transform: translateX(0); opacity: 1; }
`;

const fadeIn = keyframes`
  from { opacity: 0; transform: translateY(10px); }
  to { opacity: 1; transform: translateY(0); }
`;

// Styled Components
const AnalyticsContainer = styled.div`
  margin-top: 40px;
  padding: 20px;
  border-top: 2px solid #e0e6ed;
  background: linear-gradient(135deg, #f8f9fa 0%, #e9ecef 100%);
  border-radius: 12px;
  box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
`;

const AnalyticsHeader = styled.div`
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 24px;
  flex-wrap: wrap;
  gap: 16px;
`;

const Title = styled.h2`
  margin: 0;
  color: #2c3e50;
  font-size: 1.8rem;
  font-weight: 600;
  display: flex;
  align-items: center;
  gap: 12px;

  &::before {
    content: '📊';
    font-size: 2rem;
  }
`;

const ControlPanel = styled.div`
  display: flex;
  gap: 12px;
  align-items: center;
  flex-wrap: wrap;
`;

const ActionButton = styled.button<{ $variant?: 'primary' | 'secondary' | 'success' | 'danger' }>`
  padding: 10px 16px;
  border-radius: 8px;
  border: none;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.3s ease;
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;

  ${props => {
    switch (props.$variant) {
      case 'primary':
        return `
          background: linear-gradient(135deg, #007bff 0%, #0056b3 100%);
          color: white;
          &:hover { transform: translateY(-2px); box-shadow: 0 4px 12px rgba(0, 123, 255, 0.3); }
        `;
      case 'success':
        return `
          background: linear-gradient(135deg, #28a745 0%, #1e7e34 100%);
          color: white;
          &:hover { transform: translateY(-2px); box-shadow: 0 4px 12px rgba(40, 167, 69, 0.3); }
        `;
      case 'danger':
        return `
          background: linear-gradient(135deg, #dc3545 0%, #c82333 100%);
          color: white;
          &:hover { transform: translateY(-2px); box-shadow: 0 4px 12px rgba(220, 53, 69, 0.3); }
        `;
      default:
        return `
          background: linear-gradient(135deg, #6c757d 0%, #495057 100%);
          color: white;
          &:hover { transform: translateY(-2px); box-shadow: 0 4px 12px rgba(108, 117, 125, 0.3); }
        `;
    }
  }}

  &:disabled {
    opacity: 0.6;
    cursor: not-allowed;
    transform: none !important;
  }
`;

const StatusIndicator = styled.div<{ $active: boolean }>`
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  border-radius: 20px;
  font-size: 14px;
  font-weight: 500;

  ${props =>
    props.$active
      ? css`
          background: rgba(40, 167, 69, 0.1);
          color: #28a745;
          border: 1px solid #28a745;
          animation: ${pulseGlow} 2s infinite;
        `
      : css`
          background: rgba(108, 117, 125, 0.1);
          color: #6c757d;
          border: 1px solid #6c757d;
        `}

  &::before {
    content: '${props => (props.$active ? '🟢' : '🔴')}';
    font-size: 12px;
  }
`;

const MetricsGrid = styled.div`
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
  gap: 20px;
  margin-bottom: 24px;
`;

const MetricCard = styled.div<{ $highlighted?: boolean }>`
  background: white;
  border-radius: 12px;
  padding: 20px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  border: 1px solid #e9ecef;
  transition: all 0.3s ease;
  ${css`
    animation: ${fadeIn} 0.5s ease;
  `}

  ${props =>
    props.$highlighted &&
    css`
      border-color: #007bff;
      box-shadow: 0 4px 16px rgba(0, 123, 255, 0.2);
      transform: translateY(-2px);
    `}

  &:hover {
    transform: translateY(-4px);
    box-shadow: 0 6px 20px rgba(0, 0, 0, 0.15);
  }
`;

const MetricHeader = styled.div`
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
`;

const MetricTitle = styled.h3`
  margin: 0;
  color: #495057;
  font-size: 1.1rem;
  font-weight: 600;
`;

const MetricValue = styled.div<{ $color?: string }>`
  font-size: 2rem;
  font-weight: bold;
  color: ${props => props.$color || '#007bff'};
  margin-bottom: 8px;
`;

const MetricSubtext = styled.div`
  color: #6c757d;
  font-size: 0.9rem;
`;

const TimelineContainer = styled.div`
  background: white;
  border-radius: 12px;
  padding: 20px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  margin-bottom: 24px;
`;

const TimelineHeader = styled.h3`
  margin: 0 0 20px 0;
  color: #495057;
  display: flex;
  align-items: center;
  gap: 10px;

  &::before {
    content: '📈';
    font-size: 1.5rem;
  }
`;

const TimelinePeriods = styled.div`
  display: flex;
  gap: 16px;
  flex-wrap: wrap;
  margin-bottom: 20px;
`;

const PeriodButton = styled.button<{ $active: boolean }>`
  padding: 8px 16px;
  border-radius: 20px;
  border: 1px solid #dee2e6;
  background: ${props => (props.$active ? '#007bff' : 'white')};
  color: ${props => (props.$active ? 'white' : '#495057')};
  font-weight: 500;
  cursor: pointer;
  transition: all 0.3s ease;

  &:hover {
    background: ${props => (props.$active ? '#0056b3' : '#f8f9fa')};
    transform: translateY(-1px);
  }
`;

const AggregateData = styled.div`
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
  gap: 16px;
`;

const AggregateCard = styled.div`
  background: linear-gradient(135deg, #f8f9fa 0%, #e9ecef 100%);
  border-radius: 8px;
  padding: 16px;
  border: 1px solid #dee2e6;
  ${css`
    animation: ${slideInFromRight} 0.5s ease;
  `}
`;

const TrendContainer = styled.div`
  background: white;
  border-radius: 12px;
  padding: 20px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
`;

const TrendHeader = styled.h3`
  margin: 0 0 16px 0;
  color: #495057;
  display: flex;
  align-items: center;
  gap: 10px;

  &::before {
    content: '📊';
    font-size: 1.5rem;
  }
`;

const TrendItem = styled.div<{ $positive?: boolean }>`
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  margin-bottom: 8px;
  border-radius: 8px;
  background: ${props => (props.$positive ? 'rgba(40, 167, 69, 0.1)' : 'rgba(220, 53, 69, 0.1)')};
  border: 1px solid
    ${props => (props.$positive ? 'rgba(40, 167, 69, 0.2)' : 'rgba(220, 53, 69, 0.2)')};
  color: ${props => (props.$positive ? '#155724' : '#721c24')};
`;

const TrendLabel = styled.span`
  font-weight: 500;
  color: #495057;
`;

const TrendValue = styled.span<{ $positive?: boolean }>`
  font-weight: bold;
  color: ${props => (props.$positive ? '#28a745' : '#dc3545')};

  &::before {
    content: '${props => (props.$positive ? '↗️' : '↘️')}';
    margin-right: 4px;
  }
`;

// Performance Analytics Dashboard Component
interface PerformanceAnalyticsDashboardProps {
  className?: string;
}

const PerformanceAnalyticsDashboard: React.FC<PerformanceAnalyticsDashboardProps> = ({
  className
}) => {
  console.log(
    '[PerformanceAnalyticsDashboard] Component mounting/re-rendering at',
    new Date().toISOString()
  );

  const [isRunning, setIsRunning] = useState(false);
  const [selectedPeriod, setSelectedPeriod] = useState('minute');
  const [summary, setSummary] = useState<any>(null);
  const [aggregates, setAggregates] = useState<any>({});
  const [trends, setTrends] = useState<any>({});
  const [lastUpdate, setLastUpdate] = useState<Date>(new Date());

  const periods = [
    { key: 'minute', label: 'Minute', icon: '⏱️' },
    { key: 'hour', label: 'Hour', icon: '🕐' },
    { key: 'day', label: 'Day', icon: '📅' },
    { key: 'week', label: 'Week', icon: '📆' },
    { key: 'month', label: 'Month', icon: '🗓️' },
    { key: 'year', label: 'Year', icon: '📊' }
  ];

  // Fetch performance data
  const fetchPerformanceData = useCallback(() => {
    try {
      // Get current performance summary
      if (typeof (window as any).getPerformanceSummary === 'function') {
        const summaryData = (window as any).getPerformanceSummary();
        setSummary(summaryData);
      }

      // Get time-based aggregates
      if (typeof (window as any).getPerformanceAggregates === 'function') {
        const aggregateData = (window as any).getPerformanceAggregates();
        setAggregates(aggregateData);
      }

      // Get performance trends
      if (typeof (window as any).getPerformanceTrends === 'function') {
        const trendData = (window as any).getPerformanceTrends();
        setTrends(trendData);
      }

      setLastUpdate(new Date());
    } catch (error) {
      console.error('Error fetching performance data:', error);
    }
  }, []);

  // Auto-update when running
  useEffect(() => {
    if (!isRunning) return;

    fetchPerformanceData(); // Initial fetch
    const interval = setInterval(fetchPerformanceData, 3000); // Update every 3 seconds

    return () => clearInterval(interval);
  }, [isRunning, fetchPerformanceData]);

  // Start/stop monitoring
  const toggleMonitoring = () => {
    if (isRunning) {
      setIsRunning(false);
      console.log('🛑 Performance monitoring stopped');
    } else {
      setIsRunning(true);
      console.log('🚀 Performance monitoring started');
      fetchPerformanceData();
    }
  };

  // Reset counters
  const resetCounters = () => {
    if (typeof (window as any).resetPerformanceCounters === 'function') {
      const result = (window as any).resetPerformanceCounters();
      console.log('🔄 Performance counters reset:', result);
      fetchPerformanceData();
    }
  };

  // Generate full report
  const generateReport = () => {
    const report = {
      timestamp: new Date().toISOString(),
      summary,
      aggregates,
      trends
    };

    console.group('📋 Complete Performance Report');
    console.log('📊 Summary:', summary);
    console.log('📈 Aggregates:', aggregates);
    console.log('📉 Trends:', trends);
    console.groupEnd();

    // Download as JSON
    const blob = new Blob([JSON.stringify(report, null, 2)], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `performance-report-${Date.now()}.json`;
    a.click();
    URL.revokeObjectURL(url);
  };

  const formatNumber = (num: number): string => {
    if (num >= 1000000) return `${(num / 1000000).toFixed(1)}M`;
    if (num >= 1000) return `${(num / 1000).toFixed(1)}K`;
    return num.toFixed(0);
  };

  const formatPercentage = (num: number): string => {
    return `${num >= 0 ? '+' : ''}${num.toFixed(1)}%`;
  };

  return (
    <AnalyticsContainer className={className}>
      <AnalyticsHeader>
        <Title>Performance Analytics Dashboard</Title>
        <ControlPanel>
          <StatusIndicator $active={isRunning}>
            {isRunning ? 'Monitoring Active' : 'Monitoring Stopped'}
          </StatusIndicator>
          <ActionButton $variant="primary" onClick={toggleMonitoring}>
            {isRunning ? '⏸️ Stop' : '▶️ Start'} Monitoring
          </ActionButton>
          <ActionButton $variant="secondary" onClick={fetchPerformanceData}>
            🔄 Refresh Data
          </ActionButton>
          <ActionButton $variant="success" onClick={generateReport}>
            📊 Generate Report
          </ActionButton>
          <ActionButton $variant="danger" onClick={resetCounters}>
            🗑️ Reset Counters
          </ActionButton>
        </ControlPanel>
      </AnalyticsHeader>

      {summary && (
        <MetricsGrid>
          <MetricCard $highlighted>
            <MetricHeader>
              <MetricTitle>⚡ Operations/Second</MetricTitle>
            </MetricHeader>
            <MetricValue $color="#007bff">{summary.opsPerSecond?.toFixed(1) || '0.0'}</MetricValue>
            <MetricSubtext>Real-time processing rate</MetricSubtext>
          </MetricCard>

          <MetricCard>
            <MetricHeader>
              <MetricTitle>🔢 Total Operations</MetricTitle>
            </MetricHeader>
            <MetricValue $color="#28a745">{formatNumber(summary.totalOperations || 0)}</MetricValue>
            <MetricSubtext>Since last reset</MetricSubtext>
          </MetricCard>

          <MetricCard>
            <MetricHeader>
              <MetricTitle>🎯 Total Particles</MetricTitle>
            </MetricHeader>
            <MetricValue $color="#fd7e14">{formatNumber(summary.totalParticles || 0)}</MetricValue>
            <MetricSubtext>Particles processed</MetricSubtext>
          </MetricCard>

          <MetricCard>
            <MetricHeader>
              <MetricTitle>📈 Avg Particles/Op</MetricTitle>
            </MetricHeader>
            <MetricValue $color="#6f42c1">
              {summary.avgParticlesPerOp?.toFixed(0) || '0'}
            </MetricValue>
            <MetricSubtext>Processing efficiency</MetricSubtext>
          </MetricCard>
        </MetricsGrid>
      )}

      <TimelineContainer>
        <TimelineHeader>Time-Based Performance Aggregates</TimelineHeader>
        <TimelinePeriods>
          {periods.map(period => (
            <PeriodButton
              key={period.key}
              $active={selectedPeriod === period.key}
              onClick={() => setSelectedPeriod(period.key)}
            >
              {period.icon} {period.label}
            </PeriodButton>
          ))}
        </TimelinePeriods>

        {aggregates[selectedPeriod] && (
          <AggregateData>
            <AggregateCard>
              <MetricTitle>📊 Operations</MetricTitle>
              <MetricValue $color="#007bff">
                {formatNumber(aggregates[selectedPeriod].operations)}
              </MetricValue>
            </AggregateCard>
            <AggregateCard>
              <MetricTitle>🎯 Particles</MetricTitle>
              <MetricValue $color="#28a745">
                {formatNumber(aggregates[selectedPeriod].particles)}
              </MetricValue>
            </AggregateCard>
            <AggregateCard>
              <MetricTitle>🔥 Peak Ops/Sec</MetricTitle>
              <MetricValue $color="#dc3545">
                {aggregates[selectedPeriod].peakOpsPerSec?.toFixed(1)}
              </MetricValue>
            </AggregateCard>
            <AggregateCard>
              <MetricTitle>📈 Avg Ops/Sec</MetricTitle>
              <MetricValue $color="#fd7e14">
                {aggregates[selectedPeriod].avgOpsPerSec?.toFixed(1)}
              </MetricValue>
            </AggregateCard>
          </AggregateData>
        )}

        {!aggregates[selectedPeriod] && (
          <MetricSubtext style={{ textAlign: 'center', padding: '20px' }}>
            No data available for {selectedPeriod} period yet. Start monitoring to collect data.
          </MetricSubtext>
        )}
      </TimelineContainer>

      {(trends.hourOverHour || trends.dayOverDay) && (
        <TrendContainer>
          <TrendHeader>Performance Trends</TrendHeader>

          {trends.hourOverHour && (
            <TrendItem $positive={trends.hourOverHour.operationsChange >= 0}>
              <TrendLabel>⏰ Hour-over-Hour Operations</TrendLabel>
              <TrendValue $positive={trends.hourOverHour.operationsChange >= 0}>
                {formatPercentage(trends.hourOverHour.operationsChange)}
              </TrendValue>
            </TrendItem>
          )}

          {trends.hourOverHour && (
            <TrendItem $positive={trends.hourOverHour.particlesChange >= 0}>
              <TrendLabel>⏰ Hour-over-Hour Particles</TrendLabel>
              <TrendValue $positive={trends.hourOverHour.particlesChange >= 0}>
                {formatPercentage(trends.hourOverHour.particlesChange)}
              </TrendValue>
            </TrendItem>
          )}

          {trends.dayOverDay && (
            <TrendItem $positive={trends.dayOverDay.operationsChange >= 0}>
              <TrendLabel>📅 Day-over-Day Operations</TrendLabel>
              <TrendValue $positive={trends.dayOverDay.operationsChange >= 0}>
                {formatPercentage(trends.dayOverDay.operationsChange)}
              </TrendValue>
            </TrendItem>
          )}

          {trends.dayOverDay && (
            <TrendItem $positive={trends.dayOverDay.particlesChange >= 0}>
              <TrendLabel>📅 Day-over-Day Particles</TrendLabel>
              <TrendValue $positive={trends.dayOverDay.particlesChange >= 0}>
                {formatPercentage(trends.dayOverDay.particlesChange)}
              </TrendValue>
            </TrendItem>
          )}
        </TrendContainer>
      )}

      <MetricSubtext style={{ textAlign: 'center', marginTop: '16px' }}>
        Last updated: {lastUpdate.toLocaleTimeString()} • Monitoring:{' '}
        {isRunning ? '🟢 Active' : '🔴 Stopped'} • API Status:{' '}
        {typeof (window as any).getPerformanceSummary === 'function'
          ? '✅ Available'
          : '❌ Not Available'}
      </MetricSubtext>
    </AnalyticsContainer>
  );
};

export default React.memo(PerformanceAnalyticsDashboard);
