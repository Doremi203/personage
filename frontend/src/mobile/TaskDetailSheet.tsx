import { useEffect, useState } from 'react';
import { createPortal } from 'react-dom';
import {
  Calendar as CalendarIcon,
  Check,
  ChevronDown,
  Clock,
  Clock4,
  Flag,
  Loader2,
  Pencil,
  Tag,
} from 'lucide-react';
import { Pill } from './Chrome';
import {
  CATEGORY_LABELS,
  PRIORITY_LABELS,
  SANS,
  SERIF,
  T,
  type Category,
  type Priority,
} from './tokens';
import {
  ApiTaskCategory,
  type UpdateTaskPatch,
} from '../utils/taskerService';

export interface DetailTask {
  id: string;
  title: string;
  description: string;
  status: 'unplanned' | 'planned' | 'completed';
  priority: Priority;
  category: Category;
  startLabel: string;
  endLabel: string;
  startISO?: string;
  endISO?: string;
}

interface TaskDetailSheetProps {
  task: DetailTask;
  onClose: () => void;
  onComplete: () => Promise<void>;
  onPostpone: () => Promise<void>;
  onDelete:   () => Promise<void>;
  onSave:     (patch: UpdateTaskPatch) => Promise<void>;
}

type Action = 'complete' | 'postpone' | 'delete' | 'save' | null;

const CATEGORY_TO_API: Record<Category, string> = {
  work:     ApiTaskCategory.WORK,
  study:    ApiTaskCategory.STUDY,
  personal: ApiTaskCategory.PERSONAL,
};

const TIME_SLOT_MIN = 15;

function pad2(n: number): string {
  return String(n).padStart(2, '0');
}

function snapToSlot(d: Date, round: 'floor' | 'ceil'): Date {
  const out = new Date(d);
  const fn = round === 'ceil' ? Math.ceil : Math.floor;
  const minutes = fn(out.getMinutes() / TIME_SLOT_MIN) * TIME_SLOT_MIN;
  out.setMinutes(minutes, 0, 0);
  return out;
}

function isoToLocalInput(iso?: string): string {
  if (!iso) return '';
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return '';
  return `${d.getFullYear()}-${pad2(d.getMonth() + 1)}-${pad2(d.getDate())}T${pad2(d.getHours())}:${pad2(d.getMinutes())}`;
}

function localInputToISO(s: string, round: 'floor' | 'ceil'): string | undefined {
  if (!s) return undefined;
  const d = new Date(s);
  if (Number.isNaN(d.getTime())) return undefined;
  return snapToSlot(d, round).toISOString();
}

export function TaskDetailSheet({
  task,
  onClose,
  onComplete,
  onPostpone,
  onDelete,
  onSave,
}: TaskDetailSheetProps) {
  const pal = T[task.priority];
  const catPal = T[task.category];
  const isDone = task.status === 'completed';
  const statusLabel =
    task.status === 'completed' ? 'Завершено' :
    task.status === 'planned'   ? 'Запланировано' :
                                  'Без даты';

  const [running, setRunning] = useState<Action>(null);
  const [mountTarget, setMountTarget] = useState<HTMLElement | null>(null);

  const [editing, setEditing] = useState(false);
  const [title, setTitle] = useState(task.title);
  const [description, setDescription] = useState(task.description);
  const [startLocal, setStartLocal] = useState(isoToLocalInput(task.startISO));
  const [endLocal, setEndLocal] = useState(isoToLocalInput(task.endISO));
  const [category, setCategory] = useState<Category>(task.category);

  useEffect(() => {
    setMountTarget(document.getElementById('mobile-frame'));
  }, []);

  const wrap = (kind: Exclude<Action, null>, fn: () => Promise<void>) => async () => {
    if (running) return;
    setRunning(kind);
    try { await fn(); } finally { setRunning(null); }
  };

  const handleEditToggle = async () => {
    if (running) return;
    if (!editing) {
      setEditing(true);
      return;
    }
    const patch: UpdateTaskPatch = {};
    if (title !== task.title) patch.title = title;
    if (description !== task.description) patch.description = description;
    const initialStart = isoToLocalInput(task.startISO);
    const initialEnd = isoToLocalInput(task.endISO);
    if (startLocal !== initialStart) patch.startTime = localInputToISO(startLocal, 'floor');
    if (endLocal !== initialEnd) patch.endTime = localInputToISO(endLocal, 'ceil');
    if (category !== task.category) patch.category = CATEGORY_TO_API[category];

    if (Object.keys(patch).length === 0) {
      setEditing(false);
      return;
    }
    setRunning('save');
    try {
      await onSave(patch);
      setEditing(false);
    } finally {
      setRunning(null);
    }
  };

  if (!mountTarget) return null;

  const sheet = (
    <div
      className="animate-sheet-in"
      style={{
        position: 'absolute', inset: 0, zIndex: 100,
        display: 'flex', flexDirection: 'column',
        background: T.bg, color: T.ink, fontFamily: SANS,
        overflow: 'hidden',
      }}
    >
      {/* Drag handle */}
      <div style={{
        flexShrink: 0, padding: '8px 0 4px',
        display: 'flex', justifyContent: 'center',
        background: T.bg,
      }}>
        <div style={{ width: 36, height: 5, borderRadius: 99, background: T.subtleHi }} />
      </div>

      {/* Top bar */}
      <div style={{
        flexShrink: 0,
        display: 'flex', alignItems: 'center', justifyContent: 'space-between',
        padding: '6px 12px 10px',
      }}>
        <button
          type="button"
          onClick={onClose}
          style={{
            background: 'transparent', border: 'none', cursor: 'pointer',
            padding: '6px 10px', color: T.amberDp,
            fontFamily: SANS, fontSize: 16, fontWeight: 400,
            display: 'flex', alignItems: 'center', gap: 2,
          }}
        >
          <ChevronDown size={20} strokeWidth={2.2} />
          Закрыть
        </button>
        <button
          type="button"
          aria-label={editing ? 'Сохранить' : 'Редактировать'}
          onClick={() => void handleEditToggle()}
          disabled={running !== null}
          style={{
            background: editing ? T.amberFill : 'transparent',
            border: 'none',
            cursor: running ? 'default' : 'pointer',
            padding: 8, color: editing ? T.amberDp : T.ink2,
            borderRadius: 8,
            display: 'flex', alignItems: 'center', justifyContent: 'center',
            opacity: running && running !== 'save' ? 0.5 : 1,
          }}
        >
          {running === 'save'
            ? <Loader2 size={17} className="animate-spin" />
            : editing
              ? <Check size={18} strokeWidth={2.2} />
              : <Pencil size={17} strokeWidth={1.9} />}
        </button>
      </div>

      <div style={{ flex: 1, minHeight: 0, overflow: 'auto', padding: '0 18px 24px' }}>
        {/* Color strip */}
        <div style={{
          width: 36, height: 4, borderRadius: 2, background: pal.rail, marginBottom: 14,
        }} />

        {/* Title */}
        {editing ? (
          <input
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            style={{
              width: '100%', boxSizing: 'border-box',
              fontFamily: SERIF, fontSize: 30, lineHeight: 1.12, letterSpacing: -0.5,
              color: T.ink, marginBottom: 10,
              background: T.subtle, border: `1px solid ${T.amberDp}`,
              borderRadius: 10, padding: '8px 10px', outline: 'none',
            }}
          />
        ) : (
          <div style={{
            fontFamily: SERIF, fontSize: 30, lineHeight: 1.12, letterSpacing: -0.5,
            color: T.ink, marginBottom: 10,
            textDecoration: isDone ? 'line-through' : 'none',
            textDecorationColor: T.ink4,
            wordBreak: 'break-word',
          }}>{task.title}</div>
        )}

        {editing ? (
          <textarea
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            placeholder="Описание"
            rows={3}
            style={{
              width: '100%', boxSizing: 'border-box', resize: 'none',
              fontSize: 16, color: T.ink2, lineHeight: 1.5, marginBottom: 18,
              fontFamily: SANS,
              background: T.subtle, border: `1px solid ${T.hairline}`,
              borderRadius: 10, padding: '10px 12px', outline: 'none',
            }}
          />
        ) : task.description ? (
          <div style={{
            fontSize: 15, color: T.ink2, lineHeight: 1.5, marginBottom: 22,
            wordBreak: 'break-word',
          }}>{task.description}</div>
        ) : null}

        {/* Pills (view only) */}
        {!editing && (
          <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap', marginBottom: 22 }}>
            <Pill fill={pal.fill} ink={pal.ink} dot={pal.rail}>
              Приоритет: {PRIORITY_LABELS[task.priority]}
            </Pill>
            <Pill fill={catPal.fill} ink={catPal.ink} dot={catPal.rail}>
              {CATEGORY_LABELS[task.category]}
            </Pill>
            <Pill fill={isDone ? T.okFill : T.amberFill} ink={isDone ? T.ok : T.amberInk}>
              {statusLabel}
            </Pill>
          </div>
        )}

        {/* Action row (view only, not done) */}
        {!editing && !isDone && (
          <div style={{ display: 'flex', gap: 8, marginBottom: 18 }}>
            <button
              type="button"
              onClick={wrap('complete', onComplete)}
              disabled={running !== null}
              style={{
                flex: 1.5, padding: '14px 16px', borderRadius: 14,
                background: T.ink, color: T.bg,
                border: 'none', cursor: running ? 'default' : 'pointer',
                opacity: running && running !== 'complete' ? 0.5 : 1,
                fontFamily: SANS, fontSize: 15, fontWeight: 600,
                display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 6,
              }}
            >
              {running === 'complete'
                ? <Loader2 size={17} className="animate-spin" />
                : <><Check size={17} strokeWidth={2.2} />Выполнено</>}
            </button>
            <button
              type="button"
              onClick={wrap('postpone', onPostpone)}
              disabled={running !== null}
              style={{
                flex: 1, padding: '14px 12px', borderRadius: 14,
                background: T.surface, color: T.ink,
                border: `0.5px solid ${T.hairline}`,
                cursor: running ? 'default' : 'pointer',
                opacity: running && running !== 'postpone' ? 0.5 : 1,
                fontFamily: SANS, fontSize: 15, fontWeight: 500,
                display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 6,
              }}
            >
              {running === 'postpone'
                ? <Loader2 size={16} className="animate-spin" />
                : <><Clock4 size={16} strokeWidth={1.9} />Отложить</>}
            </button>
          </div>
        )}

        {/* Properties / Editor */}
        {editing ? (
          <Section>
            <EditDateRow icon={CalendarIcon} iconBg={T.amberFill} iconInk={T.amberDp}
              label="Начало" value={startLocal} onChange={setStartLocal} />
            <EditDateRow icon={Clock} iconBg={T.infoFill} iconInk={T.info}
              label="Конец" value={endLocal} onChange={setEndLocal} />
            <CategoryPicker icon={Tag} value={category} onChange={setCategory} last />
          </Section>
        ) : (
          <Section>
            <Row icon={CalendarIcon} iconBg={T.amberFill} iconInk={T.amberDp}
              label="Начало" value={task.startLabel} />
            <Row icon={Clock} iconBg={T.infoFill} iconInk={T.info}
              label="Конец" value={task.endLabel} />
            <Row icon={Flag} iconBg={pal.fill} iconInk={pal.ink}
              label="Приоритет" value={PRIORITY_LABELS[task.priority]} dot={pal.rail} last />
          </Section>
        )}

        {/* Delete */}
        <button
          type="button"
          onClick={wrap('delete', onDelete)}
          disabled={running !== null}
          style={{
            width: '100%', padding: '14px', borderRadius: 14,
            background: T.surface, color: T.danger,
            border: `0.5px solid ${T.hairline}`,
            cursor: running ? 'default' : 'pointer',
            opacity: running && running !== 'delete' ? 0.5 : 1,
            fontFamily: SANS, fontSize: 15, fontWeight: 500,
            display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 6,
          }}
        >
          {running === 'delete' ? <Loader2 size={16} className="animate-spin" /> : 'Удалить задачу'}
        </button>
      </div>
    </div>
  );

  return createPortal(sheet, mountTarget);
}

function Section({ children }: { children: React.ReactNode }) {
  return (
    <div style={{
      background: T.surface, borderRadius: 14,
      border: `0.5px solid ${T.hairline}`,
      marginBottom: 22, overflow: 'hidden',
    }}>{children}</div>
  );
}

interface RowProps {
  icon: typeof CalendarIcon;
  iconBg: string;
  iconInk: string;
  label: string;
  value: string;
  dot?: string;
  last?: boolean;
}

function Row({ icon: Icon, iconBg, iconInk, label, value, dot, last }: RowProps) {
  return (
    <div style={{
      display: 'flex', alignItems: 'center', gap: 12,
      padding: '12px 16px',
      borderBottom: last ? 'none' : `0.5px solid ${T.hairline}`,
    }}>
      <div style={{
        width: 30, height: 30, borderRadius: 8, background: iconBg, color: iconInk,
        display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0,
      }}>
        <Icon size={15} strokeWidth={1.8} />
      </div>
      <div style={{ flex: 1, fontSize: 14, color: T.ink, fontWeight: 500 }}>{label}</div>
      <div style={{
        fontSize: 14, color: T.ink2,
        display: 'flex', alignItems: 'center', gap: 6,
      }}>
        {dot && <span style={{ width: 7, height: 7, borderRadius: '50%', background: dot }} />}
        {value}
      </div>
    </div>
  );
}

interface EditDateRowProps {
  icon: typeof CalendarIcon;
  iconBg: string;
  iconInk: string;
  label: string;
  value: string;
  onChange: (v: string) => void;
  last?: boolean;
}

function EditDateRow({ icon: Icon, iconBg, iconInk, label, value, onChange, last }: EditDateRowProps) {
  return (
    <div style={{
      display: 'flex', alignItems: 'center', gap: 12,
      padding: '10px 16px',
      borderBottom: last ? 'none' : `0.5px solid ${T.hairline}`,
    }}>
      <div style={{
        width: 30, height: 30, borderRadius: 8, background: iconBg, color: iconInk,
        display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0,
      }}>
        <Icon size={15} strokeWidth={1.8} />
      </div>
      <div style={{ flex: '0 0 auto', fontSize: 14, color: T.ink, fontWeight: 500 }}>{label}</div>
      <input
        type="datetime-local"
        value={value}
        step={TIME_SLOT_MIN * 60}
        onChange={(e) => onChange(e.target.value)}
        style={{
          flex: 1, minWidth: 0, textAlign: 'right',
          background: 'transparent', border: 'none', outline: 'none',
          fontFamily: SANS, fontSize: 16, color: T.amberDp, fontWeight: 500,
        }}
      />
    </div>
  );
}

interface CategoryPickerProps {
  icon: typeof CalendarIcon;
  value: Category;
  onChange: (v: Category) => void;
  last?: boolean;
}

function CategoryPicker({ icon: Icon, value, onChange, last }: CategoryPickerProps) {
  const cats: { id: Category; label: string }[] = [
    { id: 'work',     label: 'Работа' },
    { id: 'study',    label: 'Учёба' },
    { id: 'personal', label: 'Личное' },
  ];
  return (
    <div style={{
      display: 'flex', alignItems: 'center', gap: 12,
      padding: '10px 16px',
      borderBottom: last ? 'none' : `0.5px solid ${T.hairline}`,
    }}>
      <div style={{
        width: 30, height: 30, borderRadius: 8, background: T.subtle, color: T.ink2,
        display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0,
      }}>
        <Icon size={15} strokeWidth={1.8} />
      </div>
      <div style={{ fontSize: 14, color: T.ink, fontWeight: 500, marginRight: 'auto' }}>Категория</div>
      <div style={{ display: 'flex', gap: 4 }}>
        {cats.map((c) => {
          const sel = value === c.id;
          const pal = T[c.id];
          return (
            <button
              key={c.id}
              type="button"
              onClick={() => onChange(c.id)}
              style={{
                padding: '5px 10px', borderRadius: 999,
                background: sel ? pal.fill : 'transparent',
                color: sel ? pal.ink : T.ink3,
                border: `1px solid ${sel ? pal.rail : T.hairline}`,
                cursor: 'pointer',
                fontFamily: SANS, fontSize: 12, fontWeight: 500,
              }}
            >{c.label}</button>
          );
        })}
      </div>
    </div>
  );
}
