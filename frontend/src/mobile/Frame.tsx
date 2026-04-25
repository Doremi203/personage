import type { ReactNode } from 'react';
import { T } from './tokens';

interface FrameProps {
  children: ReactNode;
}

// Centers the mobile app in a max-width column on wider viewports.
// On phones the inner column fills the screen; on desktop it sits in the
// middle on top of the warm `bgDeep` background.
export function Frame({ children }: FrameProps) {
  return (
    <div style={{
      minHeight: '100vh',
      background: T.bgDeep,
      display: 'flex',
      justifyContent: 'center',
      alignItems: 'stretch',
    }}>
      <div
        id="mobile-frame"
        style={{
          width: '100%',
          maxWidth: 480,
          minHeight: '100vh',
          height: '100vh',
          background: T.bg,
          boxShadow: '0 0 40px -10px rgba(40,30,20,0.18)',
          display: 'flex',
          flexDirection: 'column',
          position: 'relative',
          overflow: 'hidden',
        }}
      >
        {children}
      </div>
    </div>
  );
}
