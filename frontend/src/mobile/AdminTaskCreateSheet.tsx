import { useState, type ChangeEvent } from 'react';
import {
  createAdminTask,
  type AdminCreateTaskPayload,
  type AdminTaskItem,
} from '../utils/adminService';
import { SANS, SERIF, T } from './tokens';

interface AdminTaskCreateSheetProps {
  userId: string;
  onClose: () => void;
  onCreated: (task: AdminTaskItem) => void;
}

const STATUS_OPTIONS = [
  { value: 'unplanned', label: 'Без даты' },
  { value: 'planned', label: 'Запланировано' },
  { value: 'completed', label: 'Завершено' },
];

const CATEGORY_OPTIONS = [
  { value: 'work', label: 'Работа' },
  { value: 'study', label: 'Учёба' },
  { value: 'personal', label: 'Личное' },
];

function localInputToIso(value: string): string | undefined {
  if (!value) return undefined;
  const d = new Date(value);
  return Number.isNaN(d.getTime()) ? undefined : d.toISOString();
}

export function AdminTaskCreateSheet({ userId, onClose, onCreated }: AdminTaskCreateSheetProps) {
  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [durationMinutes, setDurationMinutes] = useState(30);
  const [priority, setPriority] = useState(5);
  const [startLocal, setStartLocal] = useState('');
  const [endLocal, setEndLocal] = useState('');
  const [deadlineLocal, setDeadlineLocal] = useState('');
  const [status, setStatus] = useState('unplanned');
  const [category, setCategory] = useState('personal');
  const [isApproved, setIsApproved] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleCreate = async () => {
    if (!title.trim()) {
      setError('Укажите заголовок');
      return;
    }

    const payload: AdminCreateTaskPayload = {
      title: title.trim(),
      description,
      durationMinutes,
      priority,
      status,
      category,
      isApproved,
      startTime: localInputToIso(startLocal),
      endTime: localInputToIso(endLocal),
      deadline: localInputToIso(deadlineLocal),
    };

    setSaving(true);
    setError(null);
    try {
      const created = await createAdminTask(userId, payload);
      onCreated(created);
      onClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Не удалось создать задачу');
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
          maxWidth: 640,
          maxHeight: '90vh',
          overflow: 'auto',
          fontFamily: SANS,
          display: 'flex',
          flexDirection: 'column',
          gap: 14,
        }}
      >
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <div style={{ fontFamily: SERIF, fontSize: 24, letterSpacing: -0.4 }}>Новая задача</div>
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

        <LabeledField label="Заголовок">
          <input
            value={title}
            onChange={(event: ChangeEvent<HTMLInputElement>) => setTitle(event.target.value)}
            style={inputStyle()}
          />
        </LabeledField>

        <LabeledField label="Описание">
          <textarea
            value={description}
            onChange={(event: ChangeEvent<HTMLTextAreaElement>) => setDescription(event.target.value)}
            rows={3}
            style={{ ...inputStyle(), resize: 'vertical', fontFamily: SANS }}
          />
        </LabeledField>

        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: 12 }}>
          <LabeledField label="Начало">
            <input
              type="datetime-local"
              value={startLocal}
              onChange={(event) => setStartLocal(event.target.value)}
              style={inputStyle()}
            />
          </LabeledField>
          <LabeledField label="Конец">
            <input
              type="datetime-local"
              value={endLocal}
              onChange={(event) => setEndLocal(event.target.value)}
              style={inputStyle()}
            />
          </LabeledField>
          <LabeledField label="Дедлайн">
            <input
              type="datetime-local"
              value={deadlineLocal}
              onChange={(event) => setDeadlineLocal(event.target.value)}
              style={inputStyle()}
            />
          </LabeledField>
        </div>

        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
          <LabeledField label="Приоритет (1–10)">
            <input
              type="number"
              min={1}
              max={10}
              value={priority}
              onChange={(event) => setPriority(Number(event.target.value) || 1)}
              style={inputStyle()}
            />
          </LabeledField>
          <LabeledField label="Длительность (мин)">
            <input
              type="number"
              min={0}
              value={durationMinutes}
              onChange={(event) => setDurationMinutes(Math.max(0, Number(event.target.value) || 0))}
              style={inputStyle()}
            />
          </LabeledField>
        </div>

        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
          <LabeledField label="Статус">
            <select
              value={status}
              onChange={(event) => setStatus(event.target.value)}
              style={inputStyle()}
            >
              {STATUS_OPTIONS.map((opt) => (
                <option key={opt.value} value={opt.value}>
                  {opt.label}
                </option>
              ))}
            </select>
          </LabeledField>
          <LabeledField label="Категория">
            <select
              value={category}
              onChange={(event) => setCategory(event.target.value)}
              style={inputStyle()}
            >
              {CATEGORY_OPTIONS.map((opt) => (
                <option key={opt.value} value={opt.value}>
                  {opt.label}
                </option>
              ))}
            </select>
          </LabeledField>
        </div>

        <label style={{ display: 'flex', alignItems: 'center', gap: 10, cursor: 'pointer' }}>
          <input
            type="checkbox"
            checked={isApproved}
            onChange={(event) => setIsApproved(event.target.checked)}
            style={{ width: 16, height: 16, cursor: 'pointer' }}
          />
          <span style={{ fontSize: 13, color: T.ink2 }}>
            Одобрена (видна пользователю; снимите для ручной модерации)
          </span>
        </label>

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
            onClick={() => void handleCreate()}
            disabled={saving}
            style={{
              padding: '10px 16px',
              borderRadius: 10,
              border: 'none',
              background: T.ink,
              color: T.bg,
              fontSize: 14,
              fontWeight: 600,
              cursor: saving ? 'default' : 'pointer',
              opacity: saving ? 0.6 : 1,
              fontFamily: SANS,
            }}
          >
            {saving ? 'Создание…' : 'Создать'}
          </button>
        </div>
      </div>
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
