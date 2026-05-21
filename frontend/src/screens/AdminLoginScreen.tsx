import { useState, type FormEvent } from 'react';
import { setAdminKey } from '../utils/adminService';
import { SANS, SERIF, T } from '../mobile/tokens';

export function AdminLoginScreen() {
  const [key, setKey] = useState('');

  const handleSubmit = (event: FormEvent) => {
    event.preventDefault();
    const trimmed = key.trim();
    if (!trimmed) return;
    setAdminKey(trimmed);
  };

  return (
    <div
      style={{
        minHeight: '100vh',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        background: T.bgDeep,
        fontFamily: SANS,
        padding: 24,
      }}
    >
      <form
        onSubmit={handleSubmit}
        style={{
          background: T.surface,
          border: `0.5px solid ${T.hairline}`,
          borderRadius: 16,
          padding: 28,
          width: '100%',
          maxWidth: 380,
          display: 'flex',
          flexDirection: 'column',
          gap: 16,
        }}
      >
        <div
          style={{
            fontFamily: SERIF,
            fontSize: 26,
            color: T.ink,
            letterSpacing: -0.5,
          }}
        >
          Админка Personage
        </div>
        <div style={{ fontSize: 14, color: T.ink2, lineHeight: 1.45 }}>
          Введите ключ доступа.
        </div>
        <input
          type="password"
          autoFocus
          value={key}
          onChange={(event) => setKey(event.target.value)}
          placeholder="X-Admin-Key"
          style={{
            padding: '12px 14px',
            borderRadius: 10,
            border: `1px solid ${T.hairline}`,
            background: T.subtle,
            fontSize: 14,
            color: T.ink,
            outline: 'none',
            fontFamily: SANS,
          }}
        />
        <button
          type="submit"
          disabled={key.trim() === ''}
          style={{
            padding: '12px 14px',
            borderRadius: 10,
            border: 'none',
            background: T.ink,
            color: T.bg,
            fontSize: 14,
            fontWeight: 600,
            cursor: key.trim() === '' ? 'default' : 'pointer',
            opacity: key.trim() === '' ? 0.5 : 1,
            fontFamily: SANS,
          }}
        >
          Войти
        </button>
      </form>
    </div>
  );
}
