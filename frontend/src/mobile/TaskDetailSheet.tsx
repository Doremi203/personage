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
  MoreHorizontal,
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

export interface DetailTask {
  id: string;
  title: string;
  description: string;
  status: 'unplanned' | 'planned' | 'completed';
  priority: Priority;
  category: Category;
  startLabel: string;
  endLabel: string;
}

interface TaskDetailSheetProps {
  task: DetailTask;
  onClose: () => void;
  onComplete: () => Promise<void>;
  onPostpone: () => Promise<void>;
  onDelete:   () => Promise<void>;
}

type Action = 'complete' | 'postpone' | 'delete' | null;

export function TaskDetailSheet({
  task,
  onClose,
  onComplete,
  onPostpone,
  onDelete,
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

  useEffect(() => {
    setMountTarget(document.getElementById('mobile-frame'));
  }, []);

  const wrap = (kind: Exclude<Action, null>, fn: () => Promise<void>) => async () => {
    if (running) return;
    setRunning(kind);
    try { await fn(); } finally { setRunning(null); }
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
          aria-label="Ещё"
          style={{
            background: 'transparent', border: 'none', cursor: 'pointer',
            padding: 8, color: T.ink2,
          }}
        >
          <MoreHorizontal size={20} strokeWidth={1.8} />
        </button>
      </div>

      <div style={{ flex: 1, minHeight: 0, overflow: 'auto', padding: '0 18px 24px' }}>
        {/* Color strip */}
        <div style={{
          width: 36, height: 4, borderRadius: 2, background: pal.rail, marginBottom: 14,
        }} />

        {/* Title */}
        <div style={{
          fontFamily: SERIF, fontSize: 30, lineHeight: 1.12, letterSpacing: -0.5,
          color: T.ink, marginBottom: 10,
          textDecoration: isDone ? 'line-through' : 'none',
          textDecorationColor: T.ink4,
          wordBreak: 'break-word',
        }}>{task.title}</div>

        {task.description && (
          <div style={{
            fontSize: 15, color: T.ink2, lineHeight: 1.5, marginBottom: 22,
            wordBreak: 'break-word',
          }}>{task.description}</div>
        )}

        {/* Pills */}
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

        {/* Action row */}
        {!isDone && (
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

        {/* Properties */}
        <Section>
          <Row icon={CalendarIcon} iconBg={T.amberFill} iconInk={T.amberDp}
            label="Начало" value={task.startLabel} />
          <Row icon={Clock} iconBg={T.infoFill} iconInk={T.info}
            label="Конец" value={task.endLabel} />
          <Row icon={Flag} iconBg={pal.fill} iconInk={pal.ink}
            label="Приоритет" value={PRIORITY_LABELS[task.priority]} dot={pal.rail} last />
        </Section>

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
