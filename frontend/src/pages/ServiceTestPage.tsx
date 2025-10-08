import React, { useState } from 'react';
import serviceRegistration from '../../config/service_registration.json';
import { validateFields } from '../utils/validateFields';
import JsonViewer from '../components/JsonViewer';

// Helper function to get input type based on field type
const getInputType = (fieldType: string): string => {
  if (fieldType.includes('int') || fieldType.includes('float') || fieldType.includes('double')) {
    return 'number';
  }
  if (fieldType.includes('bool')) {
    return 'checkbox';
  }
  if (fieldType.includes('Timestamp') || fieldType.includes('Date')) {
    return 'datetime-local';
  }
  if (fieldType.includes('repeated') || fieldType.includes('array')) {
    return 'textarea';
  }
  return 'text';
};

// Helper function to format field value
const formatFieldValue = (value: any, fieldType: string): any => {
  if (fieldType.includes('repeated') || fieldType.includes('array')) {
    if (typeof value === 'string') {
      try {
        return JSON.parse(value);
      } catch {
        return value.split(',').map((item: string) => item.trim());
      }
    }
    return value;
  }
  if (fieldType.includes('bool')) {
    return Boolean(value);
  }
  if (fieldType.includes('int')) {
    return parseInt(value) || 0;
  }
  if (fieldType.includes('float') || fieldType.includes('double')) {
    return parseFloat(value) || 0;
  }
  return value;
};

// Helper function to generate sample data based on field type
const generateSampleData = (fieldType: string, fieldName: string): any => {
  if (fieldType.includes('repeated') || fieldType.includes('array')) {
    return ['sample_item_1', 'sample_item_2'];
  }
  if (fieldType.includes('bool')) {
    return true;
  }
  if (fieldType.includes('int')) {
    return 123;
  }
  if (fieldType.includes('float') || fieldType.includes('double')) {
    return 123.45;
  }
  if (fieldType.includes('Timestamp') || fieldType.includes('Date')) {
    return new Date().toISOString().slice(0, 16);
  }
  if (fieldName.includes('id')) {
    return 'sample_id_123';
  }
  if (fieldName.includes('email')) {
    return 'user@example.com';
  }
  if (fieldName.includes('url')) {
    return 'https://example.com';
  }
  return 'sample_value';
};

interface ServiceTestPageProps {
  serviceName: string;
}

const ServiceTestPage: React.FC<ServiceTestPageProps> = ({ serviceName }) => {
  const service = serviceRegistration.find((s: any) => s.name === serviceName);
  const [payloads, setPayloads] = useState<Record<string, any>>({});
  const [results, setResults] = useState<Record<string, any>>({});
  const [errors, setErrors] = useState<Record<string, string>>({});

  if (!service) {
    return <div>Service not found: {serviceName}</div>;
  }

  const actionMap = service.action_map ?? {};
  const actions = Object.keys(actionMap);

  // Standard frontend → backend JSON envelope contract
  const envelopeExample = {
    type: 'service:action:v1:requested',
    payload: {
      // action-specific fields go here
      // Example: field1: "value1", field2: 123, field3: true
    },
    metadata: {
      global_context: {
        user_id: 'guest_123',
        campaign_id: '0',
        correlation_id: 'corr_1699999999999',
        session_id: 'session_abc',
        device_id: 'device_abc',
        source: 'frontend'
      },
      envelope_version: '1.0.0',
      environment: 'development',
      ServiceSpecific: {
        [serviceName]: {
          action: 'action_name'
        }
      }
    }
  } as const;

  const handleChange = (action: string, field: string, value: any, fieldType: string) => {
    const formattedValue = formatFieldValue(value, fieldType);
    setPayloads(prev => ({
      ...prev,
      [action]: {
        ...(prev[action] || {}),
        [field]: formattedValue
      }
    }));
  };

  const handleSampleData = (action: string, field: string, fieldType: string) => {
    const sampleValue = generateSampleData(fieldType, field);
    handleChange(action, field, sampleValue, fieldType);
  };

  const handleClearAction = (action: string) => {
    setPayloads(prev => ({
      ...prev,
      [action]: {}
    }));
    setResults(prev => ({
      ...prev,
      [action]: null
    }));
    setErrors(prev => ({
      ...prev,
      [action]: ''
    }));
  };

  const handleSubmit = async (e: React.FormEvent, action: string) => {
    e.preventDefault();
    setErrors(prev => ({ ...prev, [action]: '' }));
    setResults(prev => ({ ...prev, [action]: null }));

    const actionObj = (actionMap as Record<string, any>)[action];
    // Ensure metadata is always treated as required for envelope consistency
    const baseRequired = Array.isArray(actionObj?.rest_required_fields)
      ? actionObj.rest_required_fields
      : [];
    const requiredFields: string[] = Array.from(new Set(['metadata', ...baseRequired]));
    const payload = payloads[action] || {};

    const missingFields = validateFields(payload, requiredFields);
    if (missingFields.length > 0) {
      setErrors(prev => ({
        ...prev,
        [action]: `Missing required fields: ${missingFields.join(', ')}`
      }));
      return;
    }

    const envelope = {
      ...envelopeExample,
      type: `${service.name}:${action}:v1:requested`,
      payload: payload,
      metadata: {
        ...envelopeExample.metadata,
        ServiceSpecific: { [service.name]: { action } }
      }
    };

    setResults(prev => ({
      ...prev,
      [action]: {
        status: 'success',
        message: 'API envelope constructed successfully.',
        envelope,
        service,
        action
      }
    }));
  };

  return (
    <div className="minimal-section">
      <div className="minimal-title" style={{ textTransform: 'uppercase' }}>
        {serviceName} Service Test Page
      </div>
      <div
        className="minimal-card"
        style={{
          background: '#121212',
          border: '1px solid #333',
          padding: '16px',
          borderRadius: '8px',
          marginBottom: '24px'
        }}
      >
        <div className="minimal-card-title" style={{ fontSize: 16, marginBottom: 8 }}>
          📡 Event Envelope Structure (Frontend → Backend)
        </div>
        <p className="minimal-text" style={{ marginBottom: 12 }}>
          The frontend sends a standard JSON envelope with metadata and payload. Fill in the action
          fields below to test the service.
        </p>
        <div style={{ marginBottom: '16px' }}>
          <div className="minimal-text" style={{ fontSize: 14, marginBottom: 8, color: '#0ff' }}>
            <strong>Envelope Structure:</strong>
          </div>
          <ul style={{ color: '#ccc', fontSize: '13px', marginLeft: '20px' }}>
            <li>
              <strong>type:</strong> Service action identifier (auto-generated)
            </li>
            <li>
              <strong>payload:</strong> Action-specific data (you fill this in)
            </li>
            <li>
              <strong>metadata:</strong> Global context and service-specific info (auto-populated)
            </li>
          </ul>
        </div>
        <JsonViewer data={envelopeExample} />
      </div>
      <div
        className="minimal-card"
        style={{
          background: '#1c1c1c',
          border: '1px solid #333',
          padding: '16px',
          borderRadius: '8px',
          marginBottom: '24px'
        }}
      >
        <div className="minimal-card-title" style={{ fontSize: 16, marginBottom: 8 }}>
          Service Details
        </div>
        <p className="minimal-text">
          <b>Version:</b> {service.version}
        </p>
        <p className="minimal-text">
          <b>Proto Path:</b> {service.schema?.proto_path || 'N/A'}
        </p>
        <p className="minimal-text">
          <b>Capabilities:</b> {service.capabilities?.join(', ') || 'None'}
        </p>
        <p className="minimal-text">
          <b>Dependencies:</b> {service.dependencies?.join(', ') || 'None'}
        </p>
      </div>

      {actions.map(action => {
        const actionObj = (actionMap as Record<string, any>)[action];
        const allFields = actionObj?.fields || {};
        const baseRequired = Array.isArray(actionObj?.rest_required_fields)
          ? actionObj.rest_required_fields
          : [];
        const requiredFields: string[] = Array.from(new Set(['metadata', ...baseRequired]));
        const payload = payloads[action] || {};
        const result = results[action];
        const error = errors[action];

        return (
          <div
            key={action}
            style={{
              background: '#181818',
              border: '1px solid #555',
              borderRadius: '8px',
              padding: '16px',
              marginBottom: '24px'
            }}
          >
            <div
              className="minimal-title"
              style={{ fontSize: 18, marginBottom: 16, color: '#0ff' }}
            >
              Action: {action}
            </div>

            <div style={{ marginBottom: '16px' }}>
              <div className="minimal-title" style={{ fontSize: 16, marginBottom: 8 }}>
                📋 Action Schema & Field Information
              </div>
              <div
                style={{
                  display: 'grid',
                  gridTemplateColumns: '1fr 1fr',
                  gap: '16px',
                  marginBottom: '16px'
                }}
              >
                <div>
                  <div
                    className="minimal-text"
                    style={{ fontSize: 14, marginBottom: 8, color: '#0ff' }}
                  >
                    <strong>Action Details:</strong>
                  </div>
                  <div style={{ fontSize: '13px', color: '#ccc' }}>
                    <div>
                      <strong>Proto Method:</strong> {actionObj?.proto_method || 'N/A'}
                    </div>
                    <div>
                      <strong>Request Model:</strong> {actionObj?.request_model || 'N/A'}
                    </div>
                    <div>
                      <strong>Response Model:</strong> {actionObj?.response_model || 'N/A'}
                    </div>
                    <div>
                      <strong>Required Fields:</strong>{' '}
                      {requiredFields.filter(f => f !== 'metadata').join(', ') || 'None'}
                    </div>
                  </div>
                </div>
                <div>
                  <div
                    className="minimal-text"
                    style={{ fontSize: 14, marginBottom: 8, color: '#0ff' }}
                  >
                    <strong>Field Types:</strong>
                  </div>
                  <div style={{ fontSize: '13px', color: '#ccc' }}>
                    {Object.entries(allFields).map(([fieldName, fieldMeta]: [string, any]) => (
                      <div key={fieldName}>
                        <strong>{fieldName}:</strong> {fieldMeta.type}{' '}
                        {fieldMeta.required ? '(required)' : '(optional)'}
                      </div>
                    ))}
                  </div>
                </div>
              </div>
              <details style={{ marginTop: '12px' }}>
                <summary style={{ cursor: 'pointer', color: '#0ff', fontSize: '14px' }}>
                  🔍 View Raw Schema JSON
                </summary>
                <div style={{ marginTop: '8px' }}>
                  <JsonViewer
                    data={{
                      action,
                      proto_method: actionObj?.proto_method,
                      request_model: actionObj?.request_model,
                      response_model: actionObj?.response_model,
                      required_fields: requiredFields,
                      fields: allFields
                    }}
                  />
                </div>
              </details>
            </div>

            {Object.keys(allFields).length > 0 && (
              <form onSubmit={e => handleSubmit(e, action)} style={{ marginTop: '24px' }}>
                <div
                  style={{
                    display: 'flex',
                    justifyContent: 'space-between',
                    alignItems: 'center',
                    marginBottom: '16px'
                  }}
                >
                  <div className="minimal-title" style={{ fontSize: 14, color: '#0ff' }}>
                    🧪 Test Payload Builder
                  </div>
                  <div>
                    <button
                      type="button"
                      onClick={() => handleClearAction(action)}
                      className="minimal-button"
                      style={{
                        fontSize: 12,
                        background: '#333',
                        color: '#ff6b6b',
                        border: '1px solid #ff6b6b',
                        borderRadius: 4,
                        marginRight: 8,
                        padding: '4px 8px'
                      }}
                    >
                      Clear All
                    </button>
                    <button
                      type="button"
                      onClick={() => {
                        Object.entries(allFields).forEach(
                          ([fieldName, fieldMeta]: [string, any]) => {
                            handleSampleData(action, fieldName, fieldMeta.type);
                          }
                        );
                      }}
                      className="minimal-button"
                      style={{
                        fontSize: 12,
                        background: '#333',
                        color: '#4ecdc4',
                        border: '1px solid #4ecdc4',
                        borderRadius: 4,
                        padding: '4px 8px'
                      }}
                    >
                      Fill Sample Data
                    </button>
                  </div>
                </div>

                <div style={{ display: 'grid', gap: '12px' }}>
                  {Object.entries(allFields).map(([fieldName, fieldMeta]: [string, any]) => {
                    const inputType = getInputType(fieldMeta.type);
                    const isRequired = requiredFields.includes(fieldName);
                    const currentValue = payload[fieldName];

                    return (
                      <div
                        key={fieldName}
                        style={{
                          display: 'flex',
                          alignItems: inputType === 'checkbox' ? 'center' : 'flex-start',
                          gap: '12px',
                          padding: '12px',
                          background: '#1a1a1a',
                          border: '1px solid #333',
                          borderRadius: '6px'
                        }}
                      >
                        <div style={{ minWidth: '150px', flexShrink: 0 }}>
                          <label
                            className="minimal-text"
                            htmlFor={`${action}-${fieldName}`}
                            style={{ fontSize: 13, color: '#fff', display: 'block' }}
                          >
                            {fieldName}
                            {isRequired && (
                              <span style={{ color: '#ff6b6b', marginLeft: 4 }}>*</span>
                            )}
                          </label>
                          <div style={{ fontSize: '11px', color: '#888', marginTop: '2px' }}>
                            {fieldMeta.type}
                          </div>
                        </div>

                        <div style={{ flex: 1, display: 'flex', alignItems: 'center', gap: '8px' }}>
                          {inputType === 'checkbox' ? (
                            <input
                              id={`${action}-${fieldName}`}
                              type="checkbox"
                              checked={Boolean(currentValue)}
                              onChange={e =>
                                handleChange(action, fieldName, e.target.checked, fieldMeta.type)
                              }
                              style={{
                                width: '16px',
                                height: '16px',
                                accentColor: '#0ff'
                              }}
                            />
                          ) : inputType === 'textarea' ? (
                            <textarea
                              id={`${action}-${fieldName}`}
                              value={
                                typeof currentValue === 'string'
                                  ? currentValue
                                  : JSON.stringify(currentValue || [], null, 2)
                              }
                              onChange={e =>
                                handleChange(action, fieldName, e.target.value, fieldMeta.type)
                              }
                              style={{
                                width: '100%',
                                minHeight: '80px',
                                fontSize: '13px',
                                background: '#222',
                                color: '#0ff',
                                border: '1px solid #0ff',
                                borderRadius: '4px',
                                padding: '8px',
                                fontFamily: 'monospace'
                              }}
                              placeholder={`Enter ${fieldMeta.type.includes('repeated') ? 'JSON array' : 'JSON object'}...`}
                            />
                          ) : (
                            <input
                              id={`${action}-${fieldName}`}
                              type={inputType}
                              value={currentValue || ''}
                              onChange={e =>
                                handleChange(action, fieldName, e.target.value, fieldMeta.type)
                              }
                              style={{
                                width: '100%',
                                fontSize: '13px',
                                background: '#222',
                                color: '#0ff',
                                border: '1px solid #0ff',
                                borderRadius: '4px',
                                padding: '8px'
                              }}
                              placeholder={`Enter ${fieldMeta.type}...`}
                            />
                          )}

                          <button
                            type="button"
                            onClick={() => handleSampleData(action, fieldName, fieldMeta.type)}
                            className="minimal-button"
                            style={{
                              fontSize: 11,
                              background: '#333',
                              color: '#4ecdc4',
                              border: '1px solid #4ecdc4',
                              borderRadius: 4,
                              padding: '4px 6px',
                              whiteSpace: 'nowrap'
                            }}
                            title="Fill with sample data"
                          >
                            Sample
                          </button>
                        </div>
                      </div>
                    );
                  })}
                </div>

                <div style={{ marginTop: '16px', display: 'flex', gap: '8px' }}>
                  <button
                    type="submit"
                    className="minimal-button"
                    style={{
                      fontSize: 13,
                      background: '#0ff',
                      color: '#000',
                      border: '1px solid #0ff',
                      borderRadius: 4,
                      padding: '8px 16px',
                      fontWeight: 'bold'
                    }}
                  >
                    🚀 Build & Test Envelope
                  </button>
                </div>
              </form>
            )}

            {error && (
              <div
                style={{
                  marginTop: '16px',
                  padding: '12px',
                  background: '#2d1b1b',
                  border: '1px solid #ff6b6b',
                  borderRadius: '6px'
                }}
              >
                <div className="minimal-text" style={{ color: '#ff6b6b', fontSize: '14px' }}>
                  <strong>❌ Error:</strong> {error}
                </div>
              </div>
            )}

            {result && (
              <div style={{ marginTop: '24px' }}>
                <div
                  className="minimal-title"
                  style={{
                    fontSize: 16,
                    marginBottom: 12,
                    color: '#4ecdc4',
                    display: 'flex',
                    alignItems: 'center',
                    gap: '8px'
                  }}
                >
                  ✅ Test Result - Constructed Request Envelope
                </div>

                <div
                  style={{
                    background: '#1a2d1a',
                    border: '1px solid #4ecdc4',
                    borderRadius: '6px',
                    padding: '12px',
                    marginBottom: '16px'
                  }}
                >
                  <div style={{ fontSize: '13px', color: '#4ecdc4', marginBottom: '8px' }}>
                    <strong>Status:</strong> {result.status} | <strong>Message:</strong>{' '}
                    {result.message}
                  </div>
                  <div style={{ fontSize: '12px', color: '#ccc' }}>
                    <strong>Service:</strong> {result.service?.name} | <strong>Action:</strong>{' '}
                    {result.action}
                  </div>
                </div>

                <div style={{ marginBottom: '12px' }}>
                  <div
                    className="minimal-text"
                    style={{ fontSize: 14, marginBottom: 8, color: '#0ff' }}
                  >
                    <strong>📤 Complete Event Envelope:</strong>
                  </div>
                  <JsonViewer data={result.envelope} />
                </div>

                <div style={{ marginTop: '16px' }}>
                  <div
                    className="minimal-text"
                    style={{ fontSize: 14, marginBottom: 8, color: '#0ff' }}
                  >
                    <strong>📋 Payload Only (for debugging):</strong>
                  </div>
                  <JsonViewer data={result.envelope.payload} />
                </div>

                <div
                  style={{
                    marginTop: '16px',
                    padding: '12px',
                    background: '#1a1a2e',
                    border: '1px solid #0ff',
                    borderRadius: '6px',
                    fontSize: '13px',
                    color: '#ccc'
                  }}
                >
                  <div style={{ color: '#0ff', marginBottom: '8px' }}>
                    <strong>💡 Next Steps:</strong>
                  </div>
                  <ul style={{ margin: 0, paddingLeft: '20px' }}>
                    <li>Copy the envelope JSON to test with your backend API</li>
                    <li>Use the payload section to verify field mapping</li>
                    <li>Check the metadata structure for proper context</li>
                    <li>Validate required fields are present and correctly typed</li>
                  </ul>
                </div>
              </div>
            )}
          </div>
        );
      })}
    </div>
  );
};

export default ServiceTestPage;
