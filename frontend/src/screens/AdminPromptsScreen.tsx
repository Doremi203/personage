import { useCallback, useEffect, useState } from 'react';
import {
  listAdminPrompts,
  type AdminPromptItem,
} from '../utils/adminService';
import { SANS, SERIF, T } from '../mobile/tokens';
import { AdminPromptEditorSheet } from '../mobile/AdminPromptEditorSheet';

interface AdminPromptsScreenProps {
  onBack: () => void;
}

function formatDateTime(iso?: string): string {
  if (!iso) return '—';
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return '—';
  return d.toLocaleString('ru-RU', { dateStyle: 'short', timeStyle: 'short' });
}

export function AdminPromptsScreen({ onBack }: AdminPromptsScreenProps) {
  const [prompts, setPrompts] = useState<AdminPromptItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [selected, setSelected] = useState<AdminPromptItem | null>(null);

  const reload = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      setPrompts(await listAdminPrompts());
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Не удалось загрузить промпты');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void reload();
  }, [reload]);

  return (
    <div
      style={{
        minHeight: '100vh',
        background: T.bgDeep,
        fontFamily: SANS,
        padding: 24,
      }}
    >
      <div
        style={{
          maxWidth: 1080,
          margin: '0 auto',
          display: 'flex',
          flexDirection: 'column',
          gap: 16,
        }}
      >
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            gap: 12,
          }}
        >
          <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
            <button
              type="button"
              onClick={onBack}
              style={{
                padding: '8px 14px',
                borderRadius: 10,
                border: `0.5px solid ${T.hairline}`,
                background: T.surface,
                color: T.ink2,
                fontSize: 13,
                cursor: 'pointer',
                fontFamily: SANS,
              }}
            >
              ← К пользователям
            </button>
            <div style={{ fontFamily: SERIF, fontSize: 26, color: T.ink, letterSpacing: -0.5 }}>
              LLM промпты
            </div>
          </div>
          <button
            type="button"
            onClick={() => void reload()}
            style={{
              padding: '8px 14px',
              borderRadius: 10,
              border: `0.5px solid ${T.hairline}`,
              background: T.surface,
              color: T.ink2,
              fontSize: 13,
              cursor: 'pointer',
              fontFamily: SANS,
            }}
          >
            Обновить
          </button>
        </div>

        {error && (
          <div
            style={{
              padding: 14,
              background: T.dangerFill,
              color: T.danger,
              borderRadius: 10,
              fontSize: 14,
            }}
          >
            {error}
          </div>
        )}

        {loading ? (
          <div style={{ padding: 24, color: T.ink3, textAlign: 'center' }}>Загрузка…</div>
        ) : prompts.length === 0 ? (
          <div style={{ padding: 24, color: T.ink3, textAlign: 'center' }}>Промптов нет</div>
        ) : (
          <div style={{ display: 'grid', gap: 10 }}>
            {prompts.map((prompt) => (
              <div
                key={prompt.id}
                role="button"
                tabIndex={0}
                onClick={() => setSelected(prompt)}
                onKeyDown={(event) => {
                  if (event.key === 'Enter' || event.key === ' ') {
                    event.preventDefault();
                    setSelected(prompt);
                  }
                }}
                style={{
                  background: T.surface,
                  border: `0.5px solid ${T.hairline}`,
                  borderRadius: 14,
                  padding: 16,
                  display: 'flex',
                  flexDirection: 'column',
                  gap: 6,
                  cursor: 'pointer',
                }}
              >
                <div
                  style={{
                    fontSize: 12,
                    color: T.ink3,
                    fontFamily: 'ui-monospace, SFMono-Regular, Menlo, monospace',
                  }}
                >
                  {prompt.id}
                </div>
                <div style={{ fontFamily: SERIF, fontSize: 20, color: T.ink, lineHeight: 1.2 }}>
                  {prompt.description}
                </div>
                <div style={{ fontSize: 12, color: T.ink3 }}>
                  Обновлён: {formatDateTime(prompt.updatedAt)}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {selected && (
        <AdminPromptEditorSheet
          prompt={selected}
          onClose={() => setSelected(null)}
          onSaved={(updated) => {
            setPrompts((prev) => prev.map((p) => (p.id === updated.id ? updated : p)));
            setSelected(updated);
          }}
        />
      )}
    </div>
  );
}
