import { Link } from 'react-router-dom';
import serviceRegistration from '../../config/service_registration.json';

const getDummyValue = (field: string) => {
  if (field.toLowerCase().includes('id')) return '12345';
  if (field.toLowerCase().includes('name')) return 'Sample Name';
  if (field.toLowerCase().includes('email')) return 'sample@example.com';
  if (field.toLowerCase().includes('page')) return '1';
  if (field.toLowerCase().includes('size')) return '10';
  if (field.toLowerCase().includes('metadata')) return '{...}';
  return 'test';
};

const ServiceListPage: React.FC = () => {
  return (
    <div>
      <div className="minimal-section">
        <h1 className="minimal-title" style={{ fontSize: '18px' }}>
          SERVICES
        </h1>
        <p className="minimal-text">
          This is a list of all available microservices in the INOS platform.
          <br />
          Click on a service to access its dedicated test and interaction page.
        </p>
      </div>
      <div className="minimal-section">
        <div className="minimal-grid">
          {serviceRegistration.map((service: any) => {
            const actions = service.action_map ? Object.keys(service.action_map) : [];
            const firstAction = actions[0];
            const requiredFields = firstAction
              ? service.action_map[firstAction].rest_required_fields
              : [];
            return (
              <div
                key={service.name}
                className="minimal-card"
                style={{ textDecoration: 'none', minHeight: 260 }}
              >
                <div className="minimal-card-header">
                  <div
                    className="minimal-card-title"
                    style={{ textTransform: 'uppercase', fontSize: 15 }}
                  >
                    {service.name} Service
                  </div>
                  <div className="minimal-card-meta">
                    <span className="minimal-card-id">Version: {service.version}</span>
                    {service.capabilities && service.capabilities.length > 0 && (
                      <span
                        className="minimal-status active"
                        style={{ background: '#222', color: '#fff', borderColor: '#444' }}
                      >
                        {service.capabilities.join(', ')}
                      </span>
                    )}
                  </div>
                </div>
                <div className="minimal-card-description">
                  {service.schema?.proto_path && (
                    <div className="minimal-text" style={{ fontSize: 10, color: '#aaa' }}>
                      Proto: {service.schema.proto_path}
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
                          marginRight: 8
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
                      style={{ width: 'auto', marginBottom: 8 }}
                    >
                      Go to {service.name} Test Page
                    </Link>
                  </div>
                </div>
                <div style={{ marginTop: 8 }}>
                  <div className="minimal-title" style={{ fontSize: 12, marginBottom: 4 }}>
                    Actions
                  </div>
                  <ul style={{ paddingLeft: 16, marginBottom: 8 }}>
                    {actions.map(action => (
                      <li key={action} style={{ fontSize: 11, color: '#fff', marginBottom: 2 }}>
                        <span style={{ fontWeight: 'bold', color: '#0ff' }}>{action}</span>
                        {service.action_map[action].rest_required_fields && (
                          <span style={{ color: '#aaa', marginLeft: 6 }}>
                            [fields: {service.action_map[action].rest_required_fields.join(', ')}]
                          </span>
                        )}
                      </li>
                    ))}
                  </ul>
                  {firstAction && requiredFields.length > 0 && (
                    <form
                      style={{
                        background: '#111',
                        border: '1px solid #222',
                        padding: 8,
                        borderRadius: 4
                      }}
                    >
                      <div className="minimal-title" style={{ fontSize: 11, marginBottom: 4 }}>
                        <span style={{ color: '#0ff' }}>{firstAction}</span> Preview
                      </div>
                      {requiredFields.map((field: any) => (
                        <div key={field} style={{ marginBottom: 6 }}>
                          <label className="minimal-text" htmlFor={field} style={{ fontSize: 10 }}>
                            {field}:
                          </label>
                          <input
                            id={field}
                            type="text"
                            value={getDummyValue(field)}
                            readOnly
                            className="minimal-button"
                            style={{ width: 120, marginLeft: 6, fontSize: 10 }}
                          />
                        </div>
                      ))}
                      <button
                        type="button"
                        className="minimal-button"
                        style={{ fontSize: 10, marginTop: 4 }}
                        disabled
                      >
                        Submit (Preview)
                      </button>
                    </form>
                  )}
                </div>
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
};

export default ServiceListPage;
