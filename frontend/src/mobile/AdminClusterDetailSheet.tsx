import { useEffect, useState } from 'react';
import {
  listAdminClusterEvents,
  type AdminClusterEventItem,
  type AdminClusterListItem,
} from '../utils/adminService';
import { SANS, SERIF, T } from './tokens';

interface AdminClusterDetailSheetProps {
  userId: string;
  clusterId: string;
  summary: AdminClusterListItem | null;
  onClose: () => void;
  onOpenTask: (taskId: string) => void;
}

const STATUS_LABELS: Record<string, string> = {
  open: 'Открыт',
  processing: 'Обработка',
  closed: 'Закрыт',
};

const OUTCOME_LABELS: Record<string, string> = {
  task_generated: 'Задача создана',
  non_actionable: 'Не actionable',
  empty: 'Пустой',
};

function formatDateTime(iso?: string): string {
  if (!iso) return '—';
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return '—';
  return d.toLocaleString('ru-RU', { dateStyle: 'short', timeStyle: 'short' });
}

function stringifyContext(ctx: unknown): string {
  if (typeof ctx === 'string') return ctx;
  try {
    return JSON.stringify(ctx, null, 2);
  } catch {
    return String(ctx);
  }
}

export function AdminClusterDetailSheet({
  userId,
  clusterId,
  summary,
  onClose,
  onOpenTask,
}: AdminClusterDetailSheetProps) {
  const [events, setEvents] = useState<AdminClusterEventItem[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const loading = events === null && error === null;

  useEffect(() => {
    let cancelled = false;
    void listAdminClusterEvents(userId, clusterId)
      .then((items) => {
        if (!cancelled) setEvents(items);
      })
      .catch((err: unknown) => {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : 'Не удалось загрузить события');
        }
      });
    return () => {
      cancelled = true;
    };
  }, [userId, clusterId]);

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
          maxWidth: 720,
          maxHeight: '90vh',
          overflow: 'auto',
          fontFamily: SANS,
          display: 'flex',
          flexDirection: 'column',
          gap: 14,
        }}
      >
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <div style={{ fontFamily: SERIF, fontSize: 24, letterSpacing: -0.4 }}>Кластер</div>
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
            wordBreak: 'break-all',
          }}
        >
          {clusterId}
        </div>

        {summary && (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
            <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap', alignItems: 'center' }}>
              <span
                style={{
                  fontSize: 11,
                  padding: '3px 8px',
                  borderRadius: 999,
                  background: T.subtle,
                  color: T.ink2,
                }}
              >
                {STATUS_LABELS[summary.status] ?? summary.status}
              </span>
              {summary.generationOutcome && (
                <span
                  style={{
                    fontSize: 11,
                    padding: '3px 8px',
                    borderRadius: 999,
                    background:
                      summary.generationOutcome === 'task_generated' ? T.okFill : T.subtle,
                    color: summary.generationOutcome === 'task_generated' ? T.ok : T.ink2,
                  }}
                >
                  {OUTCOME_LABELS[summary.generationOutcome] ?? summary.generationOutcome}
                </span>
              )}
              <span style={{ fontSize: 11, color: T.ink3 }}>событий: {summary.eventCount}</span>
            </div>
            {summary.generationReason && (
              <div style={{ fontSize: 13, color: T.ink2, lineHeight: 1.4 }}>
                {summary.generationReason}
              </div>
            )}
            <div style={{ fontSize: 12, color: T.ink3, display: 'flex', gap: 14, flexWrap: 'wrap' }}>
              <span>создан: {formatDateTime(summary.createdAt)}</span>
              <span>обновлён: {formatDateTime(summary.updatedAt)}</span>
            </div>
          </div>
        )}

        {summary?.taskId && (
          <button
            type="button"
            onClick={() => onOpenTask(summary.taskId!)}
            style={{
              padding: '10px 16px',
              borderRadius: 10,
              border: 'none',
              background: T.amberDp,
              color: T.bg,
              fontSize: 14,
              fontWeight: 600,
              cursor: 'pointer',
              fontFamily: SANS,
              alignSelf: 'flex-start',
            }}
          >
            Перейти к задаче →
          </button>
        )}

        <div
          style={{
            marginTop: 6,
            fontSize: 12,
            color: T.ink3,
            textTransform: 'uppercase',
            letterSpacing: 0.4,
          }}
        >
          События
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

        {loading ? (
          <div style={{ padding: 14, color: T.ink3, textAlign: 'center' }}>Загрузка…</div>
        ) : events && events.length === 0 ? (
          <div style={{ padding: 14, color: T.ink3, textAlign: 'center' }}>Событий нет</div>
        ) : (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
            {(events ?? []).map((event) => (
              <div
                key={event.id}
                style={{
                  background: T.surface,
                  border: `0.5px solid ${T.hairline}`,
                  borderRadius: 12,
                  padding: 12,
                  display: 'flex',
                  flexDirection: 'column',
                  gap: 6,
                }}
              >
                <div style={{ display: 'flex', gap: 10, flexWrap: 'wrap', alignItems: 'center' }}>
                  <span
                    style={{
                      fontSize: 11,
                      padding: '3px 8px',
                      borderRadius: 999,
                      background: T.subtle,
                      color: T.ink2,
                    }}
                  >
                    {event.source}
                  </span>
                  <span style={{ fontSize: 12, color: T.ink3 }}>
                    {formatDateTime(event.occurredAt)}
                  </span>
                  <span
                    style={{
                      fontSize: 11,
                      color: T.ink4,
                      fontFamily: 'ui-monospace, SFMono-Regular, Menlo, monospace',
                    }}
                  >
                    {event.id}
                  </span>
                </div>
                <pre
                  style={{
                    margin: 0,
                    fontSize: 12,
                    color: T.ink,
                    fontFamily: 'ui-monospace, SFMono-Regular, Menlo, monospace',
                    whiteSpace: 'pre-wrap',
                    wordBreak: 'break-word',
                    background: T.subtle,
                    padding: 10,
                    borderRadius: 8,
                  }}
                >
                  {stringifyContext(event.context)}
                </pre>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
