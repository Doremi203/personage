import { useCallback, useEffect, useState } from 'react';
import {
  listAdminUserNotifications,
  type AdminNotificationItem,
} from '../utils/adminService';
import { SANS, SERIF, T } from '../mobile/tokens';

interface AdminUserNotificationsTabProps {
  userId: string;
}

const TYPE_LABELS: Record<string, string> = {
  admin: 'Админ',
  upcoming_event: 'Скорое событие',
  schedule_change: 'Изменение расписания',
};

function formatDateTime(iso?: string): string {
  if (!iso) return '—';
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return '—';
  return d.toLocaleString('ru-RU', { dateStyle: 'short', timeStyle: 'short' });
}

export function AdminUserNotificationsTab({ userId }: AdminUserNotificationsTabProps) {
  const [notifications, setNotifications] = useState<AdminNotificationItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const reload = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      setNotifications(await listAdminUserNotifications(userId));
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Не удалось загрузить уведомления');
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
      ) : notifications.length === 0 ? (
        <div style={{ padding: 24, color: T.ink3, textAlign: 'center' }}>Уведомлений нет</div>
      ) : (
        <div style={{ display: 'grid', gap: 10 }}>
          {notifications.map((n) => {
            const isRead = Boolean(n.readAt);
            return (
              <div
                key={n.id}
                style={{
                  background: T.surface,
                  border: `0.5px solid ${isRead ? T.hairline : T.amberDp}`,
                  borderRadius: 14,
                  padding: 16,
                  display: 'flex',
                  flexDirection: 'column',
                  gap: 6,
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
                    {TYPE_LABELS[n.type] ?? n.type}
                  </span>
                  {!isRead && (
                    <span
                      style={{
                        fontSize: 11,
                        padding: '3px 8px',
                        borderRadius: 999,
                        background: T.amberFill,
                        color: T.amberInk,
                        fontWeight: 600,
                        textTransform: 'uppercase',
                        letterSpacing: 0.4,
                      }}
                    >
                      Не прочитано
                    </span>
                  )}
                </div>
                <div
                  style={{
                    fontFamily: SERIF,
                    fontSize: 18,
                    color: T.ink,
                    lineHeight: 1.2,
                  }}
                >
                  {n.title || '(без заголовка)'}
                </div>
                {n.text && (
                  <div style={{ fontSize: 14, color: T.ink2, lineHeight: 1.4, whiteSpace: 'pre-wrap' }}>
                    {n.text}
                  </div>
                )}
                <div
                  style={{
                    fontSize: 12,
                    color: T.ink3,
                    display: 'flex',
                    gap: 14,
                    flexWrap: 'wrap',
                  }}
                >
                  <span>отправлено: {formatDateTime(n.sentAt)}</span>
                  {n.readAt && <span>прочитано: {formatDateTime(n.readAt)}</span>}
                </div>
                <div
                  style={{
                    fontFamily: 'ui-monospace, SFMono-Regular, Menlo, monospace',
                    fontSize: 11,
                    color: T.ink3,
                    wordBreak: 'break-all',
                  }}
                >
                  {n.id}
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
