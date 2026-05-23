import { useCallback, useEffect, useState } from 'react';
import {
  approveAdminTask,
  listAdminTasks,
  type AdminTaskItem,
} from '../utils/adminService';
import { SANS, SERIF, T } from '../mobile/tokens';
import { AdminTaskDetailSheet } from '../mobile/AdminTaskDetailSheet';
import { AdminSendPushSheet } from '../mobile/AdminSendPushSheet';
import { AdminUserClustersTab } from './AdminUserClustersTab';
import { AdminUserNotificationsTab } from './AdminUserNotificationsTab';
import { AdminClusterDetailSheet } from '../mobile/AdminClusterDetailSheet';

export type AdminUserTab = 'tasks' | 'clusters' | 'notifications';

const TAB_ORDER: AdminUserTab[] = ['tasks', 'clusters', 'notifications'];

const TAB_LABELS: Record<AdminUserTab, string> = {
  tasks: 'Задачи',
  clusters: 'Кластеры',
  notifications: 'Уведомления',
};

interface AdminUserTasksScreenProps {
  userId: string;
  activeTab: AdminUserTab;
  onBack: () => void;
  onChangeTab: (tab: AdminUserTab) => void;
}

function formatDateTime(iso?: string): string {
  if (!iso) return '—';
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return '—';
  return d.toLocaleString('ru-RU', { dateStyle: 'short', timeStyle: 'short' });
}

const STATUS_LABELS: Record<string, string> = {
  unplanned: 'Без даты',
  planned: 'Запланировано',
  completed: 'Завершено',
};

const CATEGORY_LABELS: Record<string, string> = {
  work: 'Работа',
  study: 'Учёба',
  personal: 'Личное',
};

export function AdminUserTasksScreen({ userId, activeTab, onBack, onChangeTab }: AdminUserTasksScreenProps) {
  const [tasks, setTasks] = useState<AdminTaskItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [selected, setSelected] = useState<AdminTaskItem | null>(null);
  const [pushOpen, setPushOpen] = useState(false);
  const [taskForCluster, setTaskForCluster] = useState<{ id: string; taskId: string } | null>(null);

  const reload = useCallback(async () => {
    if (activeTab !== 'tasks') return;
    setLoading(true);
    setError(null);
    try {
      setTasks(await listAdminTasks(userId));
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Не удалось загрузить задачи');
    } finally {
      setLoading(false);
    }
  }, [userId, activeTab]);

  useEffect(() => {
    if (activeTab === 'tasks') {
      void reload();
    }
  }, [reload, activeTab]);

  const openTaskFromCluster = useCallback(
    async (taskId: string) => {
      const existing = tasks.find((t) => t.id === taskId);
      if (existing) {
        setSelected(existing);
        return;
      }
      try {
        const list = activeTab === 'tasks' ? tasks : await listAdminTasks(userId);
        if (activeTab !== 'tasks') {
          setTasks(list);
        }
        const found = list.find((t) => t.id === taskId);
        if (found) {
          setSelected(found);
        } else {
          setError('Задача не найдена');
        }
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Не удалось открыть задачу');
      }
    },
    [tasks, userId, activeTab],
  );

  const handleQuickApprove = async (task: AdminTaskItem) => {
    try {
      const updated = await approveAdminTask(userId, task.id);
      setTasks((prev) => prev.map((t) => (t.id === task.id ? updated : t)));
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Не удалось аппрувнуть задачу');
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
            <div>
              <div style={{ fontFamily: SERIF, fontSize: 26, color: T.ink, letterSpacing: -0.5 }}>
                Задачи пользователя
              </div>
              <div
                style={{
                  fontSize: 12,
                  color: T.ink3,
                  fontFamily: 'ui-monospace, SFMono-Regular, Menlo, monospace',
                  marginTop: 2,
                }}
              >
                {userId}
              </div>
            </div>
          </div>
          <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
            <button
              type="button"
              onClick={() => setPushOpen(true)}
              style={{
                padding: '8px 14px',
                borderRadius: 10,
                border: 'none',
                background: T.ink,
                color: T.bg,
                fontSize: 13,
                fontWeight: 600,
                cursor: 'pointer',
                fontFamily: SANS,
              }}
            >
              Отправить пуш
            </button>
            {activeTab === 'tasks' && (
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
            )}
          </div>
        </div>

        <div
          style={{
            display: 'flex',
            gap: 4,
            padding: 4,
            borderRadius: 12,
            background: T.surface,
            border: `0.5px solid ${T.hairline}`,
            alignSelf: 'flex-start',
          }}
        >
          {TAB_ORDER.map((tab) => {
            const active = tab === activeTab;
            return (
              <button
                key={tab}
                type="button"
                onClick={() => onChangeTab(tab)}
                style={{
                  padding: '8px 16px',
                  borderRadius: 8,
                  border: 'none',
                  background: active ? T.ink : 'transparent',
                  color: active ? T.bg : T.ink2,
                  fontSize: 13,
                  fontWeight: active ? 600 : 500,
                  cursor: 'pointer',
                  fontFamily: SANS,
                }}
              >
                {TAB_LABELS[tab]}
              </button>
            );
          })}
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

        {activeTab === 'clusters' && (
          <AdminUserClustersTab
            userId={userId}
            onOpenTask={(taskId) => void openTaskFromCluster(taskId)}
          />
        )}

        {activeTab === 'notifications' && (
          <AdminUserNotificationsTab userId={userId} />
        )}

        {activeTab === 'tasks' && (loading ? (
          <div style={{ padding: 24, color: T.ink3, textAlign: 'center' }}>Загрузка…</div>
        ) : tasks.length === 0 ? (
          <div style={{ padding: 24, color: T.ink3, textAlign: 'center' }}>Задач нет</div>
        ) : (
          <div style={{ display: 'grid', gap: 10 }}>
            {tasks.map((task) => (
              <div
                key={task.id}
                style={{
                  background: T.surface,
                  border: `0.5px solid ${task.isApproved ? T.hairline : T.amberDp}`,
                  borderRadius: 14,
                  padding: 16,
                  display: 'grid',
                  gridTemplateColumns: '1fr auto',
                  alignItems: 'start',
                  gap: 16,
                }}
              >
                <div
                  role="button"
                  tabIndex={0}
                  onClick={() => setSelected(task)}
                  onKeyDown={(event) => {
                    if (event.key === 'Enter' || event.key === ' ') {
                      event.preventDefault();
                      setSelected(task);
                    }
                  }}
                  style={{ cursor: 'pointer', display: 'flex', flexDirection: 'column', gap: 6 }}
                >
                  <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap', alignItems: 'center' }}>
                    {!task.isApproved && (
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
                        Pending approval
                      </span>
                    )}
                    <span
                      style={{
                        fontSize: 11,
                        padding: '3px 8px',
                        borderRadius: 999,
                        background: T.subtle,
                        color: T.ink2,
                      }}
                    >
                      {STATUS_LABELS[task.status] ?? task.status}
                    </span>
                    <span
                      style={{
                        fontSize: 11,
                        padding: '3px 8px',
                        borderRadius: 999,
                        background: T.subtle,
                        color: T.ink2,
                      }}
                    >
                      {CATEGORY_LABELS[task.category] ?? task.category}
                    </span>
                    <span style={{ fontSize: 11, color: T.ink3 }}>
                      приоритет: {task.priority}
                    </span>
                  </div>
                  <div
                    style={{
                      fontFamily: SERIF,
                      fontSize: 20,
                      color: T.ink,
                      lineHeight: 1.2,
                    }}
                  >
                    {task.title || '(без названия)'}
                  </div>
                  {task.description && (
                    <div style={{ fontSize: 14, color: T.ink2, lineHeight: 1.4 }}>
                      {task.description}
                    </div>
                  )}
                  <div style={{ fontSize: 12, color: T.ink3, display: 'flex', gap: 14, flexWrap: 'wrap' }}>
                    <span>создана: {formatDateTime(task.createdAt)}</span>
                    {task.startTime && <span>старт: {formatDateTime(task.startTime)}</span>}
                    {task.endTime && <span>конец: {formatDateTime(task.endTime)}</span>}
                    {task.deadline && <span>дедлайн: {formatDateTime(task.deadline)}</span>}
                  </div>
                </div>
                <div style={{ display: 'flex', flexDirection: 'column', gap: 6, alignItems: 'flex-end' }}>
                  {!task.isApproved && (
                    <button
                      type="button"
                      onClick={(event) => {
                        event.stopPropagation();
                        void handleQuickApprove(task);
                      }}
                      style={{
                        padding: '8px 14px',
                        borderRadius: 10,
                        border: 'none',
                        background: T.ok,
                        color: T.bg,
                        fontSize: 13,
                        fontWeight: 600,
                        cursor: 'pointer',
                        fontFamily: SANS,
                      }}
                    >
                      Approve
                    </button>
                  )}
                  <button
                    type="button"
                    onClick={() => setSelected(task)}
                    style={{
                      padding: '8px 14px',
                      borderRadius: 10,
                      border: `0.5px solid ${T.hairline}`,
                      background: T.surface,
                      color: T.ink,
                      fontSize: 13,
                      cursor: 'pointer',
                      fontFamily: SANS,
                    }}
                  >
                    Открыть
                  </button>
                </div>
              </div>
            ))}
          </div>
        ))}
      </div>

      {selected && (
        <AdminTaskDetailSheet
          userId={userId}
          task={selected}
          onClose={() => setSelected(null)}
          onSaved={(updated) => {
            setTasks((prev) => prev.map((t) => (t.id === updated.id ? updated : t)));
            setSelected(updated);
          }}
          onOpenCluster={
            selected.clusterId
              ? (clusterId) => {
                  setTaskForCluster({ id: clusterId, taskId: selected.id });
                  setSelected(null);
                }
              : undefined
          }
        />
      )}

      {taskForCluster && (
        <AdminClusterDetailSheet
          userId={userId}
          clusterId={taskForCluster.id}
          summary={null}
          onClose={() => setTaskForCluster(null)}
          onOpenTask={(taskId) => {
            setTaskForCluster(null);
            void openTaskFromCluster(taskId);
          }}
        />
      )}

      {pushOpen && (
        <AdminSendPushSheet userId={userId} onClose={() => setPushOpen(false)} />
      )}
    </div>
  );
}
