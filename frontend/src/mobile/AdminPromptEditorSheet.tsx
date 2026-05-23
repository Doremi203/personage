import { useState } from 'react';
import {
  updateAdminPrompt,
  type AdminPromptItem,
  type AdminPromptUpdate,
} from '../utils/adminService';
import { SANS, SERIF, T } from './tokens';

interface AdminPromptEditorSheetProps {
  prompt: AdminPromptItem;
  onClose: () => void;
  onSaved: (prompt: AdminPromptItem) => void;
}

export function AdminPromptEditorSheet({ prompt, onClose, onSaved }: AdminPromptEditorSheetProps) {
  const [systemTemplate, setSystemTemplate] = useState(prompt.systemTemplate);
  const [userTemplate, setUserTemplate] = useState(prompt.userTemplate);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const dirty =
    systemTemplate !== prompt.systemTemplate || userTemplate !== prompt.userTemplate;

  const handleSave = async () => {
    if (!dirty) {
      onClose();
      return;
    }
    const patch: AdminPromptUpdate = {};
    if (systemTemplate !== prompt.systemTemplate) patch.systemTemplate = systemTemplate;
    if (userTemplate !== prompt.userTemplate) patch.userTemplate = userTemplate;

    setSaving(true);
    setError(null);
    try {
      const updated = await updateAdminPrompt(prompt.id, patch);
      onSaved(updated);
      onClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Не удалось сохранить промпт');
    } finally {
      setSaving(false);
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
      <div
        onClick={(event) => event.stopPropagation()}
        style={{
          background: T.bg,
          color: T.ink,
          borderRadius: 18,
          padding: 24,
          width: '100%',
          maxWidth: 880,
          maxHeight: '92vh',
          overflow: 'auto',
          fontFamily: SANS,
          display: 'flex',
          flexDirection: 'column',
          gap: 14,
        }}
      >
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
            <div style={{ fontFamily: SERIF, fontSize: 24, letterSpacing: -0.4 }}>
              {prompt.description}
            </div>
            <div
              style={{
                fontSize: 12,
                color: T.ink3,
                fontFamily: 'ui-monospace, SFMono-Regular, Menlo, monospace',
              }}
            >
              {prompt.id}
            </div>
          </div>
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

        <label style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
          <span style={{ fontSize: 12, color: T.ink3, textTransform: 'uppercase', letterSpacing: 0.4 }}>
            System template
          </span>
          <textarea
            value={systemTemplate}
            onChange={(event) => setSystemTemplate(event.target.value)}
            rows={18}
            style={{
              padding: '12px 14px',
              borderRadius: 10,
              border: `1px solid ${T.hairline}`,
              background: T.subtle,
              fontSize: 13,
              color: T.ink,
              outline: 'none',
              fontFamily: 'ui-monospace, SFMono-Regular, Menlo, monospace',
              resize: 'vertical',
              width: '100%',
              boxSizing: 'border-box',
              lineHeight: 1.5,
            }}
          />
        </label>

        <label style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
          <span style={{ fontSize: 12, color: T.ink3, textTransform: 'uppercase', letterSpacing: 0.4 }}>
            User template
          </span>
          <textarea
            value={userTemplate}
            onChange={(event) => setUserTemplate(event.target.value)}
            rows={4}
            style={{
              padding: '12px 14px',
              borderRadius: 10,
              border: `1px solid ${T.hairline}`,
              background: T.subtle,
              fontSize: 13,
              color: T.ink,
              outline: 'none',
              fontFamily: 'ui-monospace, SFMono-Regular, Menlo, monospace',
              resize: 'vertical',
              width: '100%',
              boxSizing: 'border-box',
              lineHeight: 1.5,
            }}
          />
        </label>

        <div style={{ fontSize: 12, color: T.ink3 }}>
          Подстановки используют Go-шаблоны: <code>{'{{.events}}'}</code>,{' '}
          <code>{'{{.owner_emails}}'}</code>, <code>{'{{.user_name}}'}</code>. Изменения подхватываются tasker в течение 30 секунд.
        </div>

        <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 10, marginTop: 8 }}>
          <button
            type="button"
            onClick={onClose}
            style={{
              padding: '10px 16px',
              borderRadius: 10,
              border: `0.5px solid ${T.hairline}`,
              background: T.surface,
              color: T.ink,
              fontSize: 14,
              cursor: 'pointer',
              fontFamily: SANS,
            }}
          >
            Отмена
          </button>
          <button
            type="button"
            onClick={() => void handleSave()}
            disabled={saving || !dirty}
            style={{
              padding: '10px 16px',
              borderRadius: 10,
              border: 'none',
              background: T.ink,
              color: T.bg,
              fontSize: 14,
              fontWeight: 600,
              cursor: saving || !dirty ? 'default' : 'pointer',
              opacity: saving || !dirty ? 0.6 : 1,
              fontFamily: SANS,
            }}
          >
            {saving ? 'Сохранение…' : 'Сохранить'}
          </button>
        </div>
      </div>
    </div>
  );
}
