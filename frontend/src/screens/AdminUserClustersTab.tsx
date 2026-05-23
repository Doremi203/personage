import { useCallback, useEffect, useState } from 'react';
import {
  listAdminClusters,
  type AdminClusterListItem,
} from '../utils/adminService';
import { SANS, T } from '../mobile/tokens';
import { AdminClusterDetailSheet } from '../mobile/AdminClusterDetailSheet';

interface AdminUserClustersTabProps {
  userId: string;
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

export function AdminUserClustersTab({ userId, onOpenTask }: AdminUserClustersTabProps) {
  const [clusters, setClusters] = useState<AdminClusterListItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [selected, setSelected] = useState<AdminClusterListItem | null>(null);

  const reload = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      setClusters(await listAdminClusters(userId));
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Не удалось загрузить кластеры');
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
      ) : clusters.length === 0 ? (
        <div style={{ padding: 24, color: T.ink3, textAlign: 'center' }}>Кластеров нет</div>
      ) : (
        <div style={{ display: 'grid', gap: 10 }}>
          {clusters.map((cluster) => (
            <div
              key={cluster.id}
              role="button"
              tabIndex={0}
              onClick={() => setSelected(cluster)}
              onKeyDown={(event) => {
                if (event.key === 'Enter' || event.key === ' ') {
                  event.preventDefault();
                  setSelected(cluster);
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
                  {STATUS_LABELS[cluster.status] ?? cluster.status}
                </span>
                {cluster.generationOutcome && (
                  <span
                    style={{
                      fontSize: 11,
                      padding: '3px 8px',
                      borderRadius: 999,
                      background:
                        cluster.generationOutcome === 'task_generated' ? T.okFill : T.subtle,
                      color:
                        cluster.generationOutcome === 'task_generated' ? T.ok : T.ink2,
                    }}
                  >
                    {OUTCOME_LABELS[cluster.generationOutcome] ?? cluster.generationOutcome}
                  </span>
                )}
                <span style={{ fontSize: 11, color: T.ink3 }}>событий: {cluster.eventCount}</span>
                {cluster.taskId && (
                  <span
                    style={{
                      fontSize: 11,
                      padding: '3px 8px',
                      borderRadius: 999,
                      background: T.amberFill,
                      color: T.amberInk,
                      fontWeight: 600,
                    }}
                  >
                    есть задача
                  </span>
                )}
              </div>
              <div
                style={{
                  fontFamily: 'ui-monospace, SFMono-Regular, Menlo, monospace',
                  fontSize: 13,
                  color: T.ink2,
                  wordBreak: 'break-all',
                }}
              >
                {cluster.id}
              </div>
              {cluster.generationReason && (
                <div style={{ fontSize: 13, color: T.ink2, lineHeight: 1.4 }}>
                  {cluster.generationReason}
                </div>
              )}
              <div style={{ fontSize: 12, color: T.ink3, display: 'flex', gap: 14, flexWrap: 'wrap' }}>
                <span>создан: {formatDateTime(cluster.createdAt)}</span>
                <span>обновлён: {formatDateTime(cluster.updatedAt)}</span>
              </div>
            </div>
          ))}
        </div>
      )}

      {selected && (
        <AdminClusterDetailSheet
          userId={userId}
          clusterId={selected.id}
          summary={selected}
          onClose={() => setSelected(null)}
          onOpenTask={(taskId) => {
            setSelected(null);
            onOpenTask(taskId);
          }}
        />
      )}
    </div>
  );
}
