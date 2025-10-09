// web/src/ui/ErrorBoundary.tsx
import React from 'react';

export class ErrorBoundary extends React.Component<React.PropsWithChildren, {error?: Error}> {
  constructor(props:any){ super(props); this.state = {}; }
  static getDerivedStateFromError(error: Error){ return { error }; }
  componentDidCatch(error: Error, info: any){ console.error('UI crashed:', error, info); }
  render(){
    if (this.state.error) {
      return (
        <div style={{padding:16, fontFamily:'monospace'}}>
          <h3>UI crashed</h3>
          <pre>{String(this.state.error?.message || this.state.error)}</pre>
        </div>
      );
    }
    return this.props.children;
  }
}
