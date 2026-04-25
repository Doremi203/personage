import { SANS, T } from './tokens';

interface ErrorStateProps {
  message: string;
  onRetry: () => void;
}

export function ErrorState({ message, onRetry }: ErrorStateProps) {
  return (
    <div style={{ padding: '48px 32px 16px', textAlign: 'center' }}>
      <div style={{ fontSize: 14, color: T.danger, marginBottom: 12 }}>{message}</div>
      <button
        type="button"
        onClick={onRetry}
        style={{
          padding: '10px 18px', borderRadius: 12,
          background: T.ink, color: T.bg,
          border: 'none', cursor: 'pointer',
          fontFamily: SANS, fontSize: 14, fontWeight: 600,
        }}
      >Повторить</button>
    </div>
  );
}
