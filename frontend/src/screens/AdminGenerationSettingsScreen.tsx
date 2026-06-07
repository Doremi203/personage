import { useCallback, useEffect, useMemo, useState } from 'react';
import {
  getAdminGenerationSettings,
  updateAdminGenerationSettings,
  type AdminGenerationSettings,
  type AdminGenerationSettingsUpdate,
} from '../utils/adminService';
import { SANS, SERIF, T } from '../mobile/tokens';

interface AdminGenerationSettingsScreenProps {
  onBack: () => void;
}

type FieldKey =
  | 'minSimilarity'
  | 'closedSimilarityThreshold'
  | 'topK'
  | 'maxEventCount'
  | 'inactivityMinutes'
  | 'batchSize';

interface FieldDef {
  key: FieldKey;
  label: string;
  hint: string;
  kind: 'similarity' | 'integer';
  step: number;
}

const FIELDS: FieldDef[] = [
  {
    key: 'minSimilarity',
    label: 'min_similarity',
    hint: 'Порог присоединения события к кластеру (0–1).',
    kind: 'similarity',
    step: 0.01,
  },
  {
    key: 'closedSimilarityThreshold',
    label: 'closed_similarity_threshold',
    hint: 'Порог отсева дублей по закрытым кластерам (0–1).',
    kind: 'similarity',
    step: 0.01,
  },
  {
    key: 'topK',
    label: 'top_k',
    hint: 'Сколько кандидатов-кластеров искать (целое ≥ 1).',
    kind: 'integer',
    step: 1,
  },
  {
    key: 'maxEventCount',
    label: 'max_event_count',
    hint: 'Триггер закрытия кластера по числу событий (≥ 1).',
    kind: 'integer',
    step: 1,
  },
  {
    key: 'inactivityMinutes',
    label: 'inactivity_minutes',
    hint: 'Окно неактивности для закрытия кластера, минуты (≥ 1).',
    kind: 'integer',
    step: 1,
  },
  {
    key: 'batchSize',
    label: 'batch_size',
    hint: 'Сколько закрываемых кластеров обрабатывать за тик (≥ 1).',
    kind: 'integer',
    step: 1,
  },
];

function formatDateTime(iso?: string): string {
  if (!iso) return '—';
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return '—';
  return d.toLocaleString('ru-RU', { dateStyle: 'short', timeStyle: 'short' });
}

function toFormValues(s: AdminGenerationSettings): Record<FieldKey, string> {
  return {
    minSimilarity: String(s.minSimilarity),
    closedSimilarityThreshold: String(s.closedSimilarityThreshold),
    topK: String(s.topK),
    maxEventCount: String(s.maxEventCount),
    inactivityMinutes: String(s.inactivityMinutes),
    batchSize: String(s.batchSize),
  };
}

function validateField(field: FieldDef, raw: string): string | null {
  const value = Number(raw);
  if (raw.trim() === '' || Number.isNaN(value)) {
    return 'Введите число';
  }
  if (field.kind === 'similarity') {
    if (value <= 0 || value > 1) return 'Должно быть в (0, 1]';
  } else {
    if (!Number.isInteger(value)) return 'Должно быть целым';
    if (value < 1) return 'Должно быть ≥ 1';
  }
  return null;
}

export function AdminGenerationSettingsScreen({ onBack }: AdminGenerationSettingsScreenProps) {
  const [settings, setSettings] = useState<AdminGenerationSettings | null>(null);
  const [values, setValues] = useState<Record<FieldKey, string> | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const reload = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const loaded = await getAdminGenerationSettings();
      setSettings(loaded);
      setValues(toFormValues(loaded));
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Не удалось загрузить настройки');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void reload();
  }, [reload]);

  const fieldErrors = useMemo(() => {
    if (!values) return {} as Record<FieldKey, string | null>;
    const result = {} as Record<FieldKey, string | null>;
    for (const field of FIELDS) {
      result[field.key] = validateField(field, values[field.key]);
    }
    return result;
  }, [values]);

  const hasErrors = useMemo(
    () => Object.values(fieldErrors).some((e) => e !== null),
    [fieldErrors],
  );

  const changedKeys = useMemo<FieldKey[]>(() => {
    if (!settings || !values) return [];
    return FIELDS.filter((field) => Number(values[field.key]) !== settings[field.key]).map(
      (field) => field.key,
    );
  }, [settings, values]);

  const dirty = changedKeys.length > 0;

  const handleSave = async () => {
    if (!settings || !values || !dirty || hasErrors) return;
    const patch: AdminGenerationSettingsUpdate = {};
    for (const key of changedKeys) {
      patch[key] = Number(values[key]);
    }

    setSaving(true);
    setError(null);
    try {
      const updated = await updateAdminGenerationSettings(patch);
      setSettings(updated);
      setValues(toFormValues(updated));
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Не удалось сохранить настройки');
    } finally {
      setSaving(false);
    }
  };

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
          maxWidth: 720,
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
              Настройки генерации
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

        {loading || !values || !settings ? (
          <div style={{ padding: 24, color: T.ink3, textAlign: 'center' }}>Загрузка…</div>
        ) : (
          <div
            style={{
              background: T.surface,
              border: `0.5px solid ${T.hairline}`,
              borderRadius: 14,
              padding: 20,
              display: 'flex',
              flexDirection: 'column',
              gap: 18,
            }}
          >
            {FIELDS.map((field) => {
              const fieldError = fieldErrors[field.key];
              return (
                <label
                  key={field.key}
                  style={{ display: 'flex', flexDirection: 'column', gap: 6 }}
                >
                  <span
                    style={{
                      fontSize: 12,
                      color: T.ink3,
                      fontFamily: 'ui-monospace, SFMono-Regular, Menlo, monospace',
                    }}
                  >
                    {field.label}
                  </span>
                  <input
                    type="number"
                    step={field.step}
                    value={values[field.key]}
                    onChange={(event) =>
                      setValues((prev) =>
                        prev ? { ...prev, [field.key]: event.target.value } : prev,
                      )
                    }
                    style={{
                      padding: '10px 12px',
                      borderRadius: 10,
                      border: `1px solid ${fieldError ? T.danger : T.hairline}`,
                      background: T.subtle,
                      fontSize: 14,
                      color: T.ink,
                      outline: 'none',
                      fontFamily: SANS,
                      width: '100%',
                      boxSizing: 'border-box',
                    }}
                  />
                  <span style={{ fontSize: 12, color: fieldError ? T.danger : T.ink3 }}>
                    {fieldError ?? field.hint}
                  </span>
                </label>
              );
            })}

            <div style={{ fontSize: 12, color: T.ink3 }}>
              Обновлено: {formatDateTime(settings.updatedAt)}. Изменения подхватываются tasker в
              течение 30 секунд.
            </div>

            <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 10 }}>
              <button
                type="button"
                onClick={() => void handleSave()}
                disabled={saving || !dirty || hasErrors}
                style={{
                  padding: '10px 16px',
                  borderRadius: 10,
                  border: 'none',
                  background: T.ink,
                  color: T.bg,
                  fontSize: 14,
                  fontWeight: 600,
                  cursor: saving || !dirty || hasErrors ? 'default' : 'pointer',
                  opacity: saving || !dirty || hasErrors ? 0.6 : 1,
                  fontFamily: SANS,
                }}
              >
                {saving ? 'Сохранение…' : 'Сохранить'}
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
