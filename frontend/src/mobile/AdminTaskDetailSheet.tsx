import { useState, type ChangeEvent } from 'react';
import {
  approveAdminTask,
  updateAdminTask,
  type AdminTaskItem,
} from '../utils/adminService';
import type { AdminUpdateTaskPatch } from '../utils/taskerService';
import { SANS, SERIF, T } from './tokens';

interface AdminTaskDetailSheetProps {
  userId: string;
  task: AdminTaskItem;
  onClose: () => void;
  onSaved: (task: AdminTaskItem) => void;
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

function isoToLocalInput(iso?: string): string {
  if (!iso) return '';
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return '';
  const pad = (n: number) => String(n).padStart(2, '0');
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

function localInputToIso(value: string): string | undefined {
  if (!value) return undefined;
  const d = new Date(value);
  return Number.isNaN(d.getTime()) ? undefined : d.toISOString();
}

export function AdminTaskDetailSheet({ userId, task, onClose, onSaved }: AdminTaskDetailSheetProps) {
  const [title, setTitle] = useState(task.title);
  const [description, setDescription] = useState(task.description);
  const [startLocal, setStartLocal] = useState(isoToLocalInput(task.startTime));
  const [endLocal, setEndLocal] = useState(isoToLocalInput(task.endTime));
  const [deadlineLocal, setDeadlineLocal] = useState(isoToLocalInput(task.deadline));
  const [priority, setPriority] = useState(task.priority);
  const [status, setStatus] = useState(task.status);
  const [category, setCategory] = useState(task.category);
  const [saving, setSaving] = useState(false);
  const [approving, setApproving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSave = async () => {
    const patch: AdminUpdateTaskPatch = {};
    if (title !== task.title) patch.title = title;
    if (description !== task.description) patch.description = description;
    const initialStart = isoToLocalInput(task.startTime);
    const initialEnd = isoToLocalInput(task.endTime);
    const initialDeadline = isoToLocalInput(task.deadline);
    if (startLocal !== initialStart) patch.startTime = localInputToIso(startLocal);
    if (endLocal !== initialEnd) patch.endTime = localInputToIso(endLocal);
    if (deadlineLocal !== initialDeadline) patch.deadline = localInputToIso(deadlineLocal);
    if (priority !== task.priority) patch.priority = priority;
    if (status !== task.status) patch.status = status;
    if (category !== task.category) patch.category = category;

    if (Object.keys(patch).length === 0) {
      onClose();
      return;
    }

    setSaving(true);
    setError(null);
    try {
      const updated = await updateAdminTask(userId, task.id, patch);
      onSaved(updated);
      onClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Не удалось сохранить');
    } finally {
      setSaving(false);
    }
  };

  const handleApprove = async () => {
    setApproving(true);
    setError(null);
    try {
      const updated = await approveAdminTask(userId, task.id);
      onSaved(updated);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Не удалось аппрувнуть');
    } finally {
      setApproving(false);
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
          <div style={{ fontFamily: SERIF, fontSize: 24, letterSpacing: -0.4 }}>Задача</div>
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

        <div style={{ fontSize: 12, color: T.ink3, fontFamily: 'ui-monospace, SFMono-Regular, Menlo, monospace' }}>
          {task.id}
        </div>

        {!task.isApproved && (
          <div
            style={{
              padding: 12,
              borderRadius: 10,
              background: T.amberFill,
              color: T.amberInk,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
              gap: 12,
            }}
          >
            <span style={{ fontSize: 13, fontWeight: 500 }}>Задача ожидает модерации</span>
            <button
              type="button"
              onClick={() => void handleApprove()}
              disabled={approving}
              style={{
                padding: '8px 14px',
                borderRadius: 10,
                border: 'none',
                background: T.ok,
                color: T.bg,
                fontSize: 13,
                fontWeight: 600,
                cursor: approving ? 'default' : 'pointer',
                opacity: approving ? 0.6 : 1,
                fontFamily: SANS,
              }}
            >
              {approving ? '...' : 'Approve'}
            </button>
          </div>
        )}

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

        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: 12 }}>
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
            {saving ? 'Сохранение…' : 'Сохранить'}
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
