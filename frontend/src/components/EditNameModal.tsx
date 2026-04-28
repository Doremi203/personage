import { useEffect, useState, type FormEvent } from 'react';
import { createPortal } from 'react-dom';
import { Loader2 } from 'lucide-react';
import { SANS, SERIF, T } from '../mobile/tokens';

interface EditNameModalProps {
  currentName: string;
  onClose: () => void;
  onSave: (name: string) => Promise<void>;
}

const NAME_PATTERN = /^[\p{L}\s\-']+$/u;

function validateName(value: string): string | null {
  const trimmed = value.trim();
  if (!trimmed) return 'Имя не может быть пустым';
  if (trimmed.length < 2 || trimmed.length > 100) {
    return 'Имя должно содержать от 2 до 100 символов';
  }
  if (!NAME_PATTERN.test(trimmed)) {
    return 'Имя может содержать только буквы, пробелы, дефисы и апострофы';
  }
  return null;
}

export function EditNameModal({ currentName, onClose, onSave }: EditNameModalProps) {
  const [name, setName] = useState(currentName);
  const [serverError, setServerError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const [mountTarget, setMountTarget] = useState<HTMLElement | null>(null);

  useEffect(() => {
    setMountTarget(document.getElementById('mobile-frame') ?? document.body);
  }, []);

  const validation = validateName(name);
  const unchanged = name.trim() === currentName.trim();
  const submitDisabled = saving || validation !== null || unchanged;

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    if (submitDisabled) return;
    setServerError(null);
    setSaving(true);
    try {
      await onSave(name.trim());
      onClose();
    } catch (err) {
      setServerError(err instanceof Error ? err.message : 'Не удалось изменить имя');
    } finally {
      setSaving(false);
    }
  };

  if (!mountTarget) return null;

  const modal = (
    <div
      onClick={() => { if (!saving) onClose(); }}
      style={{
        position: 'absolute', inset: 0, zIndex: 100,
        background: 'rgba(0, 0, 0, 0.4)',
        display: 'flex', alignItems: 'center', justifyContent: 'center',
        padding: 20,
      }}
    >
      <form
        onClick={(e) => e.stopPropagation()}
        onSubmit={(e) => { void handleSubmit(e); }}
        style={{
          background: T.bg, borderRadius: 16,
          border: `0.5px solid ${T.hairline}`,
          padding: 20, width: '100%', maxWidth: 360,
          display: 'flex', flexDirection: 'column', gap: 14,
          fontFamily: SANS, color: T.ink,
        }}
      >
        <div style={{
          fontFamily: SERIF, fontSize: 22, lineHeight: 1.2,
          letterSpacing: -0.3,
        }}>
          Изменить имя
        </div>

        {serverError && (
          <div style={{
            padding: '10px 12px',
            fontSize: 13, color: T.danger, background: T.dangerFill,
            borderRadius: 10, lineHeight: 1.4,
          }}>{serverError}</div>
        )}

        <input
          type="text"
          autoFocus
          value={name}
          maxLength={100}
          disabled={saving}
          onChange={(e) => {
            setName(e.target.value);
            if (serverError) setServerError(null);
          }}
          placeholder="Имя"
          style={{
            padding: '12px 14px', borderRadius: 10,
            border: `0.5px solid ${T.hairline}`,
            background: T.surface, color: T.ink,
            fontFamily: SANS, fontSize: 16, outline: 'none',
          }}
        />

        {validation && !serverError && (
          <div style={{ fontSize: 12.5, color: T.ink3, lineHeight: 1.4 }}>
            {validation}
          </div>
        )}

        <div style={{ display: 'flex', gap: 10, marginTop: 4 }}>
          <button
            type="button"
            onClick={onClose}
            disabled={saving}
            style={{
              flex: 1, padding: '12px 16px', borderRadius: 12,
              background: T.surface, color: T.ink,
              border: `0.5px solid ${T.hairline}`,
              cursor: saving ? 'default' : 'pointer',
              fontFamily: SANS, fontSize: 15, fontWeight: 500,
              opacity: saving ? 0.6 : 1,
            }}
          >Отмена</button>
          <button
            type="submit"
            disabled={submitDisabled}
            style={{
              flex: 1, padding: '12px 16px', borderRadius: 12,
              background: T.ink, color: T.bg, border: 'none',
              cursor: submitDisabled ? 'default' : 'pointer',
              fontFamily: SANS, fontSize: 15, fontWeight: 600,
              opacity: submitDisabled ? 0.5 : 1,
              display: 'inline-flex', alignItems: 'center', justifyContent: 'center', gap: 6,
            }}
          >
            {saving && <Loader2 size={15} className="animate-spin" />}
            Сохранить
          </button>
        </div>
      </form>
    </div>
  );

  return createPortal(modal, mountTarget);
}
