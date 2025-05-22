import React, { useState } from 'react';

type ButtonProps = {
  text: string;
  onClick: () => void;
  color?: string;
};

// Function component with props
export function Button({ text, onClick, color = 'blue' }: ButtonProps) {
  return (
    <button 
      style={{ backgroundColor: color }}
      onClick={onClick}
    >
      {text}
    </button>
  );
}

type CardProps = {
  title: string;
  children: React.ReactNode;
};

// Arrow function component that accepts children
export const Card = ({ title, children }: CardProps) => {
  const [expanded, setExpanded] = useState(false);
  
  return (
    <div className="card">
      <h2 onClick={() => setExpanded(!expanded)}>{title}</h2>
      {expanded && <div className="card-content">{children}</div>}
    </div>
  );
};

// Unexported arrow function component
const InternalComponent = ({ message }: { message: string }) => {
  return <div className="internal">{message}</div>;
};

// Component that uses children prop implicitly via props
function Container(props: { className?: string }) {
  return (
    <div className={props.className || 'container'}>
      {props.children}
    </div>
  );
}

export default Container;