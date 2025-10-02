import React from 'react';

interface TextareaFieldProps extends React.TextareaHTMLAttributes<HTMLTextAreaElement> {
  id: string;
}

const TextareaField: React.FC<TextareaFieldProps> = ({ id, ...props }) => {
  return (
    <textarea
      id={id}
      {...props}
      className="minimal-textarea"
      style={{
        width: '100%',
        background: '#000',
        color: '#fff',
        border: '1px solid #333',
        padding: '8px',
        fontSize: '12px',
        fontFamily: 'inherit',
        minHeight: '100px',
      }}
    />
  );
};

export default TextareaField;
