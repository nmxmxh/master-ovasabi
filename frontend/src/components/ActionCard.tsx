import React from 'react';
import JsonViewer from './JsonViewer';

const getDummyValue = (field: string) => {
  if (field.toLowerCase().includes('id')) return '12345';
  if (field.toLowerCase().includes('name')) return 'Sample Name';
  if (field.toLowerCase().includes('email')) return 'sample@example.com';
  if (field.toLowerCase().includes('page')) return 1;
  if (field.toLowerCase().includes('size')) return 10;
  if (field.toLowerCase().includes('metadata')) return { client: 'test-suite' };
  return 'test';
};

interface ActionCardProps {
  serviceName: string;
  serviceVersion: string;
  actionName: string;
  actionDetails: any;
}

const ActionCard: React.FC<ActionCardProps> = ({
  serviceName,
  serviceVersion,
  actionName,
  actionDetails
}) => {
  const {
    proto_method,
    request_model,
    response_model,
    rest_required_fields = [],
    fields = {}
  } = actionDetails;

  const samplePayload = Object.keys(fields).reduce(
    (acc, field) => {
      acc[field] = getDummyValue(field);
      return acc;
    },
    {} as Record<string, any>
  );

  return (
    <div
      style={{
        flex: '1 1 320px',
        minWidth: 320,
        maxWidth: 480,
        background: '#222',
        border: '2px solid #0ff',
        borderRadius: 8,
        padding: 16,
        boxShadow: '0 1px 8px #0ff2',
        color: '#fff',
        display: 'flex',
        flexDirection: 'column',
        gap: '8px'
      }}
    >
      <div style={{ fontWeight: 'bold', color: '#0ff', fontSize: 15, marginBottom: 2 }}>
        {actionName}
      </div>
      <div style={{ fontSize: 12, color: '#aaa', marginBottom: 2 }}>
        <b>Proto Method:</b> {proto_method || 'N/A'}
      </div>
      <div style={{ fontSize: 12, color: '#aaa', marginBottom: 2 }}>
        <b>Request Model:</b> {request_model || 'N/A'}
      </div>
      <div style={{ fontSize: 12, color: '#aaa', marginBottom: 2 }}>
        <b>Response Model:</b> {response_model || 'N/A'}
      </div>
      <div style={{ fontSize: 12, color: '#aaa', marginBottom: 2 }}>
        <b>Event Type:</b>{' '}
        <span style={{ color: '#0f0', fontWeight: 'bold' }}>
          {`${serviceName}:${actionName}:${serviceVersion}:requested`}
        </span>
      </div>

      {Object.keys(fields).length > 0 && (
        <div style={{ fontSize: 12, color: '#aaa', marginTop: 4 }}>
          <b>Fields:</b>
          <ul style={{ paddingLeft: 16, marginTop: 4, listStyleType: 'disc' }}>
            {Object.entries(fields).map(([fname, fmeta]: [string, any]) => {
              const isRequired = rest_required_fields.includes(fname);
              return (
                <li
                  key={fname}
                  style={{
                    fontSize: 11,
                    color: isRequired ? '#fff' : '#aaa',
                    marginBottom: 2
                  }}
                >
                  <span style={{ fontWeight: isRequired ? 'bold' : 'normal' }}>{fname}</span>
                  <span style={{ color: '#0f0', marginLeft: 6 }}>({fmeta.type})</span>
                  {isRequired && (
                    <span style={{ color: '#f00', marginLeft: 6 }}>[required]</span>
                  )}
                </li>
              );
            })}
          </ul>
        </div>
      )}

      {Object.keys(samplePayload).length > 0 && (
        <div style={{ fontSize: 12, color: '#aaa', marginTop: 4 }}>
          <b>Sample Request Payload:</b>
          <JsonViewer data={samplePayload} />
        </div>
      )}
    </div>
  );
};

export default ActionCard;
