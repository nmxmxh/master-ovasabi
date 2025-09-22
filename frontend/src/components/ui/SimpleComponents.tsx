import React from 'react';

// Simple component props interface
interface SimpleComponentProps {
  theme?: any;
  [key: string]: any;
}

// Hero Component - Simple and clean
export const HeroComponent: React.FC<SimpleComponentProps> = ({
  title,
  subtitle,
  cta_text,
  background_image,
  theme = {}
}) => {
  return (
    <div
      style={{
        background: background_image
          ? `url(${background_image})`
          : theme.background_color || '#000',
        backgroundSize: 'cover',
        backgroundPosition: 'center',
        padding: '60px 20px',
        textAlign: 'center',
        color: theme.text_color || '#fff',
        minHeight: '400px',
        display: 'flex',
        flexDirection: 'column',
        justifyContent: 'center',
        alignItems: 'center'
      }}
    >
      <h1
        style={{
          fontSize: '32px',
          fontWeight: 'bold',
          marginBottom: '16px',
          color: theme.primary_color || '#fff'
        }}
      >
        {title || 'Welcome'}
      </h1>

      <p
        style={{
          fontSize: '18px',
          marginBottom: '32px',
          opacity: 0.9,
          maxWidth: '600px',
          lineHeight: '1.5'
        }}
      >
        {subtitle || 'Your subtitle here'}
      </p>

      {cta_text && (
        <button
          style={{
            background: theme.primary_color || '#00AB6C',
            color: theme.background_color || '#fff',
            border: 'none',
            padding: '12px 24px',
            fontSize: '16px',
            fontWeight: 'bold',
            borderRadius: '6px',
            cursor: 'pointer',
            transition: 'all 0.2s ease'
          }}
        >
          {cta_text}
        </button>
      )}
    </div>
  );
};

// Editor Component - Simple text editor
export const EditorComponent: React.FC<SimpleComponentProps> = ({
  placeholder,
  height = '400px',
  theme = {}
}) => {
  return (
    <div
      style={{
        padding: '20px',
        background: theme.background_color || '#fff',
        border: `1px solid ${theme.border_color || '#ddd'}`,
        borderRadius: '8px',
        margin: '20px 0'
      }}
    >
      <textarea
        placeholder={placeholder || 'Start writing...'}
        style={{
          width: '100%',
          height: height,
          border: 'none',
          outline: 'none',
          fontSize: '16px',
          fontFamily: theme.font_family || 'inherit',
          lineHeight: '1.6',
          color: theme.text_color || '#333',
          background: 'transparent',
          resize: 'vertical'
        }}
      />
    </div>
  );
};

// Card Component - Simple content card
export const CardComponent: React.FC<SimpleComponentProps> = ({
  title,
  excerpt,
  author,
  claps,
  theme = {}
}) => {
  return (
    <div
      style={{
        background: theme.background_color || '#fff',
        border: `1px solid ${theme.border_color || '#ddd'}`,
        borderRadius: '8px',
        padding: '20px',
        margin: '10px 0',
        boxShadow: '0 2px 4px rgba(0,0,0,0.1)',
        transition: 'all 0.2s ease',
        cursor: 'pointer'
      }}
      onMouseOver={e => {
        e.currentTarget.style.transform = 'translateY(-2px)';
        e.currentTarget.style.boxShadow = '0 4px 8px rgba(0,0,0,0.15)';
      }}
      onMouseOut={e => {
        e.currentTarget.style.transform = 'translateY(0)';
        e.currentTarget.style.boxShadow = '0 2px 4px rgba(0,0,0,0.1)';
      }}
    >
      <h3
        style={{
          fontSize: '20px',
          fontWeight: 'bold',
          marginBottom: '12px',
          color: theme.text_color || '#333'
        }}
      >
        {title || 'Card Title'}
      </h3>

      <p
        style={{
          fontSize: '14px',
          color: theme.text_color || '#666',
          marginBottom: '16px',
          lineHeight: '1.5'
        }}
      >
        {excerpt || 'Card excerpt goes here...'}
      </p>

      <div
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          fontSize: '12px',
          color: theme.text_color || '#999'
        }}
      >
        <span>{author || 'Author'}</span>
        <span>👏 {claps || '0'}</span>
      </div>
    </div>
  );
};

// Grid Component - Simple grid layout
export const GridComponent: React.FC<SimpleComponentProps> = ({
  children,
  columns = 3,
  gap = '16px'
}) => {
  return (
    <div
      style={{
        display: 'grid',
        gridTemplateColumns: `repeat(${columns}, 1fr)`,
        gap: gap,
        padding: '20px 0'
      }}
    >
      {children}
    </div>
  );
};

// Text Component - Simple text display
export const TextComponent: React.FC<SimpleComponentProps> = ({
  content,
  size = '16px',
  weight = 'normal',
  color,
  theme = {}
}) => {
  return (
    <div
      style={{
        fontSize: size,
        fontWeight: weight,
        color: color || theme.text_color || '#333',
        lineHeight: '1.6',
        margin: '10px 0'
      }}
    >
      {content || 'Text content'}
    </div>
  );
};

// Button Component - Simple button
export const ButtonComponent: React.FC<SimpleComponentProps> = ({
  text,
  onClick,
  variant = 'primary',
  theme = {}
}) => {
  const isPrimary = variant === 'primary';

  return (
    <button
      onClick={onClick}
      style={{
        background: isPrimary ? theme.primary_color || '#00AB6C' : 'transparent',
        color: isPrimary ? theme.background_color || '#fff' : theme.primary_color || '#00AB6C',
        border: `2px solid ${theme.primary_color || '#00AB6C'}`,
        padding: '10px 20px',
        fontSize: '14px',
        fontWeight: 'bold',
        borderRadius: '6px',
        cursor: 'pointer',
        transition: 'all 0.2s ease'
      }}
    >
      {text || 'Button'}
    </button>
  );
};

// Default Component - Fallback for unknown types
export const DefaultComponent: React.FC<SimpleComponentProps> = ({ type, ...props }) => {
  return (
    <div
      style={{
        padding: '20px',
        border: '2px dashed #ddd',
        borderRadius: '8px',
        textAlign: 'center',
        color: '#666',
        background: '#f9f9f9'
      }}
    >
      <h4>Component: {type || 'Unknown'}</h4>
      <p>This component type is not implemented yet.</p>
      <pre style={{ fontSize: '12px', marginTop: '10px' }}>{JSON.stringify(props, null, 2)}</pre>
    </div>
  );
};
