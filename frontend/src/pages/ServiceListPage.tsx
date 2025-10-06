import serviceRegistration from '../../config/service_registration.json';
import ServiceCard from '../components/ServiceCard';

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
        <div style={{ display: 'flex', flexDirection: 'column', gap: '32px', width: '100%' }}>
          {serviceRegistration.map((service: any) => (
            <ServiceCard key={service.name} service={service} />
          ))}
        </div>
      </div>
    </div>
  );
};

export default ServiceListPage;
