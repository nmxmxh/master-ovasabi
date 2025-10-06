import React, { useState } from 'react';
import serviceRegistration from '../../config/service_registration.json';
import { validateFields } from '../utils/validateFields';
import JsonViewer from '../components/JsonViewer';

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

  const handleChange = (action: string, field: string, value: any) => {
    setPayloads(prev => ({
      ...prev,
      [action]: {
        ...(prev[action] || {}),
        [field]: value
      }
    }));
  };

  const handleSubmit = async (e: React.FormEvent, action: string) => {
    e.preventDefault();
    setErrors(prev => ({ ...prev, [action]: '' }));
    setResults(prev => ({ ...prev, [action]: null }));

    const actionObj = (actionMap as Record<string, any>)[action];
    const requiredFields: string[] = actionObj?.rest_required_fields || [];
    const payload = payloads[action] || {};

    const missingFields = validateFields(payload, requiredFields);
    if (missingFields.length > 0) {
      setErrors(prev => ({ ...prev, [action]: `Missing required fields: ${missingFields.join(', ')}` }));
      return;
    }

    setResults(prev => ({
      ...prev,
      [action]: {
        status: 'success',
        message: 'API call simulated successfully.',
        submittedPayload: payload,
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
        const requiredFields: string[] = actionObj?.rest_required_fields || [];
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
            <div className="minimal-title" style={{ fontSize: 18, marginBottom: 16, color: '#0ff' }}>
              Action: {action}
            </div>

            <div style={{ marginBottom: '16px' }}>
              <div className="minimal-title" style={{ fontSize: 16, marginBottom: 8 }}>
                Full Action Configuration
              </div>
              <JsonViewer data={actionObj} />
            </div>

            {Object.keys(allFields).length > 0 && (
              <form onSubmit={e => handleSubmit(e, action)} style={{ marginTop: '24px' }}>
                <div className="minimal-title" style={{ fontSize: 14, marginBottom: 16, color: '#0ff' }}>
                  Test Payload
                </div>
                {Object.entries(allFields).map(([fieldName, fieldMeta]: [string, any]) => (
                  <div key={fieldName} style={{ marginBottom: 12, display: 'flex', alignItems: 'center' }}>
                    <label
                      className="minimal-text"
                      htmlFor={`${action}-${fieldName}`}
                      style={{ fontSize: 13, color: '#fff', minWidth: '150px' }}
                    >
                      {fieldName}:
                      {requiredFields.includes(fieldName) && (
                        <span style={{ color: '#f00', marginLeft: 4 }}>*</span>
                      )}
                    </label>
                    <input
                      id={`${action}-${fieldName}`}
                      type="text"
                      value={payload[fieldName] || ''}
                      onChange={e => handleChange(action, fieldName, e.target.value)}
                      className="minimal-button"
                      style={{
                        width: '250px',
                        marginLeft: '8px',
                        fontSize: '13px',
                        background: '#222',
                        color: '#0ff',
                        border: '1px solid #0ff',
                        borderRadius: '4px'
                      }}
                      placeholder={fieldMeta.type || ''}
                    />
                    <span style={{ color: '#0f0', marginLeft: 8, fontSize: 12 }}>
                      ({fieldMeta.type})
                    </span>
                  </div>
                ))}
                <button
                  type="submit"
                  className="minimal-button"
                  style={{
                    fontSize: 13,
                    background: '#222',
                    color: '#0ff',
                    border: '1px solid #0ff',
                    borderRadius: 4,
                    marginTop: 8
                  }}
                >
                  Test Action
                </button>
              </form>
            )}

            {error && (
              <div className="minimal-text" style={{ color: '#f00', marginTop: '16px' }}>
                <b>Error:</b> {error}
              </div>
            )}

            {result && (
              <div style={{ marginTop: '24px' }}>
                <div className="minimal-title">Simulated API Result</div>
                <JsonViewer data={result} />
              </div>
            )}
          </div>
        );
      })}
    </div>
  );
};

export default ServiceTestPage;
