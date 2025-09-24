import React from 'react';

const ServicePageLayout: React.FC<{ title: string; children: React.ReactNode }> = ({
  title,
  children
}) => (
  <div>
    <div className="minimal-section">
      <h1 className="minimal-title" style={{ fontSize: '18px', textTransform: 'uppercase' }}>
        {title} Service
      </h1>
      <p className="minimal-text">
        This is a dedicated page for interacting with the {title} service.
        <br />
        You can add forms, buttons, and displays here to test its functionality.
      </p>
    </div>
    <div className="minimal-section">{children}</div>
  </div>
);

const UserServicePage: React.FC = () => {
  return (
    <ServicePageLayout title="User">
      <div className="minimal-text">
        User Service controls will be here.
        <br />
        For example, a form to create a new user or a list to display existing users.
      </div>
    </ServicePageLayout>
  );
};

export default UserServicePage;
