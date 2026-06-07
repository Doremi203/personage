import { useCallback, useEffect, useState } from 'react';
import {
  listAdminUserEvents,
  type AdminClusterEventItem,
} from '../utils/adminService';
import { SANS, T } from '../mobile/tokens';
import { AdminClusterDetailSheet } from '../mobile/AdminClusterDetailSheet';

interface AdminUserEventsTabProps {
  userId: string;
  onOpenTask: (taskId: string) => void;
}

function formatDateTime(iso?: string): string {
  if (!iso) return '—';
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return '—';
  return d.toLocaleString('ru-RU', { dateStyle: 'short', timeStyle: 'short' });
}

function formatSimilarity(value: number): string {
  if (!Number.isFinite(value)) return '—';
  return `${(value * 100).toFixed(1)}%`;
}

export function AdminUserEventsTab({ userId, onOpenTask }: AdminUserEventsTabProps) {
  const [events, setEvents] = useState<AdminClusterEventItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [selectedClusterId, setSelectedClusterId] = useState<string | null>(null);

  const reload = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      setEvents(await listAdminUserEvents(userId));
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Не удалось загрузить события');
    } finally {
      setLoading(false);
    }
  }, [userId]);

  useEffect(() => {
    void reload();
  }, [reload]);

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
      <div style={{ display: 'flex', justifyContent: 'flex-end' }}>
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
      ) : events.length === 0 ? (
        <div style={{ padding: 24, color: T.ink3, textAlign: 'center' }}>Событий нет</div>
      ) : (
        <div style={{ display: 'grid', gap: 10 }}>
          {events.map((event) => (
            <div
              key={event.id}
              style={{
                background: T.surface,
                border: `0.5px solid ${T.hairline}`,
                borderRadius: 14,
                padding: 16,
                display: 'flex',
                flexDirection: 'column',
                gap: 8,
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
                <span
                  title="Сходство с центроидом кластера в момент добавления"
                  style={{
                    fontSize: 11,
                    padding: '3px 8px',
                    borderRadius: 999,
                    background: event.similarity >= 0.8 ? T.okFill : T.subtle,
                    color: event.similarity >= 0.8 ? T.ok : T.ink2,
                    fontFamily: 'ui-monospace, SFMono-Regular, Menlo, monospace',
                  }}
                >
                  sim {formatSimilarity(event.similarity)}
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

              <div style={{ display: 'flex', gap: 8, alignItems: 'center', flexWrap: 'wrap' }}>
                <span style={{ fontSize: 12, color: T.ink3 }}>кластер:</span>
                <button
                  type="button"
                  onClick={() => setSelectedClusterId(event.clusterId)}
                  style={{
                    fontSize: 12,
                    padding: '4px 10px',
                    borderRadius: 999,
                    border: `0.5px solid ${T.hairline}`,
                    background: T.subtle,
                    color: T.ink,
                    cursor: 'pointer',
                    fontFamily: 'ui-monospace, SFMono-Regular, Menlo, monospace',
                    wordBreak: 'break-all',
                  }}
                >
                  {event.clusterId} →
                </button>
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
                {event.context}
              </pre>
            </div>
          ))}
        </div>
      )}

      {selectedClusterId && (
        <AdminClusterDetailSheet
          userId={userId}
          clusterId={selectedClusterId}
          summary={null}
          onClose={() => setSelectedClusterId(null)}
          onOpenTask={(taskId) => {
            setSelectedClusterId(null);
            onOpenTask(taskId);
          }}
        />
      )}
    </div>
  );
}
