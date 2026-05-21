import { useState, type ChangeEvent, type FormEvent } from 'react';
import { sendAdminPushToUser } from '../utils/adminService';
import { SANS, SERIF, T } from './tokens';

interface AdminSendPushSheetProps {
  userId: string;
  onClose: () => void;
}

export function AdminSendPushSheet({ userId, onClose }: AdminSendPushSheetProps) {
  const [title, setTitle] = useState('');
  const [body, setBody] = useState('');
  const [url, setUrl] = useState('');
  const [sending, setSending] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [sent, setSent] = useState(false);

  const canSubmit = title.trim().length > 0 && body.trim().length > 0 && !sending;

  const handleSubmit = async (event: FormEvent) => {
    event.preventDefault();
    if (!canSubmit) return;
    setSending(true);
    setError(null);
    try {
      await sendAdminPushToUser(userId, {
        title: title.trim(),
        body: body.trim(),
        url: url.trim() || undefined,
      });
      setSent(true);
      setTimeout(onClose, 1500);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Не удалось отправить уведомление');
    } finally {
      setSending(false);
    }
  };

  return (
    <div
      style={{
        position: 'fixed',
        inset: 0,
        background: 'rgba(20, 18, 16, 0.45)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        padding: 24,
        zIndex: 200,
      }}
      onClick={onClose}
    >
      <form
        onSubmit={(event) => void handleSubmit(event)}
        onClick={(event) => event.stopPropagation()}
        style={{
          background: T.bg,
          color: T.ink,
          borderRadius: 18,
          padding: 24,
          width: '100%',
          maxWidth: 520,
          maxHeight: '90vh',
          overflow: 'auto',
          fontFamily: SANS,
          display: 'flex',
          flexDirection: 'column',
          gap: 14,
        }}
      >
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <div style={{ fontFamily: SERIF, fontSize: 24, letterSpacing: -0.4 }}>Отправить пуш</div>
          <button
            type="button"
            onClick={onClose}
            style={{
              background: 'transparent',
              border: 'none',
              color: T.ink2,
              fontSize: 14,
              cursor: 'pointer',
            }}
          >
            Закрыть
          </button>
        </div>

        <div
          style={{
            fontSize: 12,
            color: T.ink3,
            fontFamily: 'ui-monospace, SFMono-Regular, Menlo, monospace',
          }}
        >
          {userId}
        </div>

        {error && (
          <div
            style={{
              padding: 12,
              borderRadius: 10,
              background: T.dangerFill,
              color: T.danger,
              fontSize: 13,
            }}
          >
            {error}
          </div>
        )}

        {sent && (
          <div
            style={{
              padding: 12,
              borderRadius: 10,
              background: T.amberFill,
              color: T.amberInk,
              fontSize: 13,
            }}
          >
            Отправлено
          </div>
        )}

        <LabeledField label="Заголовок">
          <input
            value={title}
            onChange={(event: ChangeEvent<HTMLInputElement>) => setTitle(event.target.value)}
            style={inputStyle()}
            autoFocus
          />
        </LabeledField>

        <LabeledField label="Текст">
          <textarea
            value={body}
            onChange={(event: ChangeEvent<HTMLTextAreaElement>) => setBody(event.target.value)}
            rows={4}
            style={{ ...inputStyle(), resize: 'vertical', fontFamily: SANS }}
          />
        </LabeledField>

        <LabeledField label="URL (необязательно)">
          <input
            value={url}
            onChange={(event: ChangeEvent<HTMLInputElement>) => setUrl(event.target.value)}
            placeholder="https://"
            style={inputStyle()}
          />
        </LabeledField>

        <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 12, marginTop: 4 }}>
          <button
            type="submit"
            disabled={!canSubmit}
            style={{
              padding: '12px 18px',
              borderRadius: 10,
              border: 'none',
              background: T.ink,
              color: T.bg,
              fontSize: 14,
              fontWeight: 600,
              cursor: canSubmit ? 'pointer' : 'default',
              opacity: canSubmit ? 1 : 0.5,
              fontFamily: SANS,
            }}
          >
            {sending ? 'Отправка…' : 'Отправить'}
          </button>
        </div>
      </form>
    </div>
  );
}

function LabeledField({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <label style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
      <span style={{ fontSize: 12, color: T.ink3, textTransform: 'uppercase', letterSpacing: 0.4 }}>
        {label}
      </span>
      {children}
    </label>
  );
}

function inputStyle(): React.CSSProperties {
  return {
    padding: '10px 12px',
    borderRadius: 10,
    border: `1px solid ${T.hairline}`,
    background: T.subtle,
    fontSize: 14,
    color: T.ink,
    outline: 'none',
    fontFamily: SANS,
    width: '100%',
    boxSizing: 'border-box',
  };
}
