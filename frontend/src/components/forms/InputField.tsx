import React from 'react';

interface InputFieldProps extends React.InputHTMLAttributes<HTMLInputElement> {
  id: string;
}

const InputField: React.FC<InputFieldProps> = ({ id, ...props }) => {
  return (
    <input
      id={id}
      {...props}
      className="minimal-input"
      style={{
        width: '100%',
        background: '#000',
        color: '#fff',
        border: '1px solid #333',
        padding: '8px',
        fontSize: '12px',
        fontFamily: 'inherit',
      }}
    />
  );
};

export default InputField;
