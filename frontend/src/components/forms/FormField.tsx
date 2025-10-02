import React from 'react';

interface FormFieldProps {
  label: string;
  htmlFor: string;
  children: React.ReactNode;
}

const FormField: React.FC<FormFieldProps> = ({ label, htmlFor, children }) => {
  return (
    <div style={{ marginBottom: '16px' }}>
      <label htmlFor={htmlFor} className="minimal-text" style={{ display: 'block', marginBottom: '4px', textTransform: 'uppercase', fontSize: '10px', fontWeight: 'bold' }}>
        {label}
      </label>
      {children}
    </div>
  );
};

export default FormField;
