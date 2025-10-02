import React from 'react';

interface SelectFieldProps extends React.SelectHTMLAttributes<HTMLSelectElement> {
  id: string;
  children: React.ReactNode;
}

const SelectField: React.FC<SelectFieldProps> = ({ id, children, ...props }) => {
  return (
    <select
      id={id}
      {...props}
      className="minimal-input" // Reuse style from input
      style={{
        width: '100%',
        background: '#000',
        color: '#fff',
        border: '1px solid #333',
        padding: '8px',
        fontSize: '12px',
        fontFamily: 'inherit',
      }}
    >
      {children}
    </select>
  );
};

export default SelectField;
