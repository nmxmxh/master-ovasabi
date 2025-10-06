import React from 'react';

interface JsonViewerProps {
  data: object;
}

const JsonViewer: React.FC<JsonViewerProps> = ({ data }) => {
  return (
    <pre
      style={{
        background: '#111',
        border: '1px solid #333',
        padding: '12px',
        borderRadius: '4px',
        color: '#0ff',
        fontSize: '12px',
        whiteSpace: 'pre-wrap',
        wordBreak: 'break-all',
        maxHeight: '400px',
        overflowY: 'auto'
      }}
    >
      <code>{JSON.stringify(data, null, 2)}</code>
    </pre>
  );
};

export default JsonViewer;
