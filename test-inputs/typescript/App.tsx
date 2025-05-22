import React from 'react';
import { Button, Card } from './Component';
import Container from './Component';

type AppProps = {
  title: string;
};

// Main app component
export default function App({ title }: AppProps) {
  return (
    <Container>
      <h1>{title}</h1>
      <Card title="Example Card">
        <p>This is a card with some content.</p>
        <Button 
          text="Click Me" 
          onClick={() => alert('Button clicked!')} 
        />
      </Card>
    </Container>
  );
}