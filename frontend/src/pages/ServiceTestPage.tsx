import React, { useState } from 'react';
import serviceRegistration from '../../config/service_registration.json';
import { getRequiredFields, validateFields } from '../utils/validateFields';

interface ServiceTestPageProps {
  serviceName: string;
}

function ServiceTestPage({ serviceName }: ServiceTestPageProps) {
  const service = serviceRegistration.find((s: any) => s.name === serviceName);
  const [action, setAction] = useState('');
  const [payload, setPayload] = useState<any>({});
  const [result, setResult] = useState<any>(null);
  const [error, setError] = useState('');

  if (!service) {
    return <div>Service not found: {serviceName}</div>;
  }

  const actions = service.action_map ? Object.keys(service.action_map) : [];
  const requiredFields = action ? getRequiredFields(serviceName, action) : [];

  const handleChange = (field: string, value: any) => {
    setPayload((prev: any) => ({ ...prev, [field]: value }));
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setResult(null);
    const missingFields = validateFields(payload, requiredFields);
    if (missingFields.length > 0) {
      setError(`Missing required fields: ${missingFields.join(', ')}`);
      return;
    }
    // Simulate API call
    setResult({ submitted: true, service: serviceName, action, payload });
  };

  return (
    <div className="minimal-section">
      <div className="minimal-title">{serviceName.toUpperCase()} SERVICE TEST</div>
      <div className="minimal-text">Select an action to test:</div>
      <select value={action} onChange={e => setAction(e.target.value)} className="minimal-button">
        <option value="">-- Select Action --</option>
        {actions.map(a => (
          <option key={a} value={a}>
            {a}
          </option>
        ))}
      </select>
      {action && (
        <form onSubmit={handleSubmit} style={{ marginTop: 16 }}>
          <div className="minimal-title">Required Fields</div>
          {requiredFields.map(field => (
            <div key={field} style={{ marginBottom: 8 }}>
              <label className="minimal-text" htmlFor={field}>
                {field}:
              </label>
              <input
                id={field}
                type="text"
                value={payload[field] || ''}
                onChange={e => handleChange(field, e.target.value)}
                className="minimal-button"
                style={{ width: 200 }}
              />
            </div>
          ))}
          <button type="submit" className="minimal-button">
            Test Action
          </button>
        </form>
      )}
      {error && (
        <div className="minimal-text" style={{ color: '#f00' }}>
          {error}
        </div>
      )}
      {result && (
        <div className="minimal-section">
          <div className="minimal-title">Result</div>
          <div className="minimal-code">{JSON.stringify(result, null, 2)}</div>
        </div>
      )}
    </div>
  );
}

export default ServiceTestPage;
