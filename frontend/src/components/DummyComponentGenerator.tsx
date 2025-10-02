import React from 'react';

interface DummyComponentGeneratorProps {
  components: Record<string, any>;
  theme: Record<string, any>;
}

const DummyComponentGenerator: React.FC<DummyComponentGeneratorProps> = ({ components, theme }) => {
  if (!components || Object.keys(components).length === 0) {
    return (
      <div style={{ padding: '20px', textAlign: 'center', color: '#888' }}>
        No UI components defined for this campaign.
      </div>
    );
  }

  return (
    <div
      style={{
        display: 'grid',
        gridTemplateColumns: 'repeat(auto-fit, minmax(250px, 1fr))',
        gap: '16px',
        padding: '20px',
        background: theme.background_color || '#0a0a0a',
      }}
    >
      {Object.entries(components).map(([name, config]) => (
        <div
          key={name}
          title={config.description}
          style={{
            background: theme.background_color || '#111',
            border: `1px solid ${theme.border_color || '#333'}`,
            borderRadius: theme.border_radius || '4px',
            padding: '16px',
            transition: 'all 0.2s ease',
            cursor: 'pointer',
            display: 'flex',
            flexDirection: 'column',
            justifyContent: 'space-between',
            minHeight: '120px',
          }}
          onMouseOver={e => {
            e.currentTarget.style.borderColor = theme.primary_color || '#555';
            e.currentTarget.style.transform = 'translateY(-2px)';
          }}
          onMouseOut={e => {
            e.currentTarget.style.borderColor = theme.border_color || '#333';
            e.currentTarget.style.transform = 'translateY(0)';
          }}
        >
          <div>
            <div
              style={{
                fontSize: '12px',
                fontWeight: 'bold',
                color: theme.primary_color || '#fff',
                marginBottom: '8px',
                textTransform: 'uppercase',
                letterSpacing: '0.5px',
              }}
            >
              {name.replace(/_/g, ' ')}
            </div>
            <div style={{ fontSize: '11px', color: '#888', marginBottom: '4px' }}>
              Type: <span style={{ color: '#aaa' }}>{config.type || 'N/A'}</span>
            </div>
          </div>
          <div style={{ fontSize: '10px', color: '#666', marginTop: '12px', paddingTop: '8px', borderTop: `1px solid ${theme.border_color || '#222'}` }}>
            {config.description || 'No description.'}
          </div>
        </div>
      ))}
    </div>
  );
};

export default DummyComponentGenerator;
