import React from 'react';
import { Link } from 'react-router-dom';

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
          <Link to="/services/user" className="minimal-card" style={{ textDecoration: 'none' }}>
            <div className="minimal-card-title" style={{ textTransform: 'uppercase' }}>
              User Service
            </div>
            <div className="minimal-card-description">Test and interact with the user service.</div>
          </Link>
        </div>
      </div>
    </div>
  );
};

export default ServiceListPage;
