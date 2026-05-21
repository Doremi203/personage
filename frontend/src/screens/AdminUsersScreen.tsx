import { useCallback, useEffect, useState } from 'react';
import {
  clearAdminKey,
  listAdminUsers,
  listModeratedUserIds,
  setUserModeration,
  type AdminUserSummary,
} from '../utils/adminService';
import { SANS, SERIF, T } from '../mobile/tokens';

interface AdminUsersScreenProps {
  onSelect: (userId: string) => void;
}

export function AdminUsersScreen({ onSelect }: AdminUsersScreenProps) {
  const [users, setUsers] = useState<AdminUserSummary[]>([]);
  const [moderated, setModerated] = useState<Set<string>>(new Set());
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [pending, setPending] = useState<Set<string>>(new Set());

  const reload = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [u, m] = await Promise.all([listAdminUsers(), listModeratedUserIds()]);
      setUsers(u);
      setModerated(new Set(m));
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Не удалось загрузить пользователей');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void reload();
  }, [reload]);

  const handleToggle = async (userId: string, current: boolean) => {
    setPending((prev) => new Set(prev).add(userId));
    setModerated((prev) => {
      const next = new Set(prev);
      if (current) next.delete(userId); else next.add(userId);
      return next;
    });
    try {
      await setUserModeration(userId, !current);
    } catch (err) {
      setModerated((prev) => {
        const next = new Set(prev);
        if (current) next.add(userId); else next.delete(userId);
        return next;
      });
      setError(err instanceof Error ? err.message : 'Не удалось обновить модерацию');
    } finally {
      setPending((prev) => {
        const next = new Set(prev);
        next.delete(userId);
        return next;
      });
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
          maxWidth: 1080,
          margin: '0 auto',
          display: 'flex',
          flexDirection: 'column',
          gap: 16,
        }}
      >
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <div style={{ fontFamily: SERIF, fontSize: 30, color: T.ink, letterSpacing: -0.5 }}>
            Пользователи
          </div>
          <button
            type="button"
            onClick={clearAdminKey}
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
            Выйти
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
        ) : users.length === 0 ? (
          <div style={{ padding: 24, color: T.ink3, textAlign: 'center' }}>Пользователей нет</div>
        ) : (
          <div
            style={{
              background: T.surface,
              border: `0.5px solid ${T.hairline}`,
              borderRadius: 14,
              overflow: 'hidden',
            }}
          >
            <div
              style={{
                display: 'grid',
                gridTemplateColumns: '320px 1fr 240px 180px',
                padding: '12px 18px',
                background: T.subtle,
                fontSize: 12,
                color: T.ink3,
                fontWeight: 500,
                textTransform: 'uppercase',
                letterSpacing: 0.4,
              }}
            >
              <span>User ID</span>
              <span>Email</span>
              <span>Имя</span>
              <span>Ручная модерация</span>
            </div>
            {users.map((u) => {
              const isModerated = moderated.has(u.id);
              const isPending = pending.has(u.id);
              return (
                <div
                  key={u.id}
                  style={{
                    display: 'grid',
                    gridTemplateColumns: '320px 1fr 240px 180px',
                    padding: '12px 18px',
                    borderTop: `0.5px solid ${T.hairline}`,
                    alignItems: 'center',
                    fontSize: 14,
                    color: T.ink,
                  }}
                >
                  <button
                    type="button"
                    onClick={() => onSelect(u.id)}
                    style={{
                      background: 'transparent',
                      border: 'none',
                      padding: 0,
                      color: T.amberDp,
                      fontFamily: 'ui-monospace, SFMono-Regular, Menlo, monospace',
                      fontSize: 12,
                      textAlign: 'left',
                      cursor: 'pointer',
                    }}
                  >
                    {u.id}
                  </button>
                  <span>{u.email}</span>
                  <span style={{ color: u.name ? T.ink : T.ink4 }}>{u.name ?? '—'}</span>
                  <label
                    style={{
                      display: 'inline-flex',
                      alignItems: 'center',
                      gap: 8,
                      cursor: isPending ? 'default' : 'pointer',
                      opacity: isPending ? 0.5 : 1,
                    }}
                  >
                    <input
                      type="checkbox"
                      checked={isModerated}
                      disabled={isPending}
                      onChange={() => void handleToggle(u.id, isModerated)}
                    />
                    <span style={{ fontSize: 13, color: T.ink2 }}>
                      {isModerated ? 'Включена' : 'Отключена'}
                    </span>
                  </label>
                </div>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
}
