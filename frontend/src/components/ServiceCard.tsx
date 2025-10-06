import React from 'react';
import { Link } from 'react-router-dom';
import ActionCard from './ActionCard';

interface ServiceCardProps {
  service: any;
}

const ServiceCard: React.FC<ServiceCardProps> = ({ service }) => {
  const actions = service.action_map ? Object.entries(service.action_map) : [];

  return (
    <div
      className="minimal-card"
      style={{
        width: '100%',
        margin: '0 auto',
        boxSizing: 'border-box',
        boxShadow: '0 2px 12px #2228',
        border: '2px solid #222',
        borderRadius: 8,
        padding: 24,
        background: '#181818'
      }}
    >
      <div className="minimal-card-header" style={{ marginBottom: 12 }}>
        <div
          className="minimal-card-title"
          style={{ textTransform: 'uppercase', fontSize: 18, letterSpacing: 1 }}
        >
          {service.name} Service
        </div>
        <div className="minimal-card-meta" style={{ marginTop: 4 }}>
          <span className="minimal-card-id" style={{ fontSize: 13, color: '#0ff' }}>
            Version: {service.version}
          </span>
          {service.capabilities && service.capabilities.length > 0 && (
            <span
              className="minimal-status active"
              style={{
                background: '#222',
                color: '#fff',
                borderColor: '#444',
                marginLeft: 12,
                padding: '2px 8px',
                borderRadius: 4,
                fontSize: 12
              }}
            >
              {service.capabilities.join(', ')}
            </span>
          )}
        </div>
      </div>
      <div className="minimal-card-description" style={{ marginBottom: 12 }}>
        {service.schema?.proto_path && (
          <div className="minimal-text" style={{ fontSize: 12, color: '#aaa', marginBottom: 2 }}>
            <b>Proto:</b> {service.schema.proto_path}
          </div>
        )}
        <div style={{ marginTop: 4 }}>
          {service.metadata_enrichment && (
            <span
              className="minimal-status"
              style={{
                background: '#0f0',
                color: '#000',
                borderColor: '#0f0',
                marginRight: 8,
                padding: '2px 8px',
                borderRadius: 4,
                fontSize: 12
              }}
            >
              Metadata Enrichment
            </span>
          )}
        </div>
        <div style={{ marginTop: 8 }}>
          <Link
            to={`/services/${service.name}`}
            className="minimal-card-button"
            style={{ width: 'auto', marginBottom: 8, fontSize: 13, padding: '4px 12px' }}
          >
            Go to {service.name} Test Page
          </Link>
        </div>
      </div>
      <div style={{ marginTop: 8 }}>
        <div className="minimal-title" style={{ fontSize: 15, marginBottom: 8, color: '#fff' }}>
          Actions
        </div>
        {actions.length === 0 ? (
          <div className="minimal-text" style={{ color: '#f00', fontSize: 12 }}>
            No actions defined for this service.
          </div>
        ) : (
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: '18px' }}>
            {actions.map(([actionName, actionDetails]) => (
              <ActionCard
                key={actionName}
                serviceName={service.name}
                serviceVersion={service.version}
                actionName={actionName}
                actionDetails={actionDetails}
              />
            ))}
          </div>
        )}
      </div>
    </div>
  );
};

export default ServiceCard;
