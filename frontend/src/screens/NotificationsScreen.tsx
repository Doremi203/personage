import { useEffect, useMemo, useState } from 'react';
import {
  BarChart2,
  Bell,
  BellOff,
  CalendarClock,
  type LucideIcon,
} from 'lucide-react';
import {
  LargeHeader,
  Segmented,
  type SegmentedItem,
} from '../mobile/Chrome';
import { SANS, SERIF, T } from '../mobile/tokens';
import {
  markAllRead,
  markRead,
  refreshNotifications,
  useNotifications,
} from '../mobile/notificationsStore';
import type { ApiNotificationItem } from '../utils/notificatorService';
import { RU_MONTHS_GEN } from '../utils/dateFormat';
import { ErrorState } from '../mobile/StateViews';

type Filter = 'all' | 'unread';
type Kind = 'reminder' | 'schedule' | 'analytics';

function kindOf(type: string): Kind {
  if (type === 'schedule_change') return 'schedule';
  if (type === 'analytics')       return 'analytics';
  return 'reminder';
}

function formatRelative(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return '';
  const now = new Date();
  const diffMs = now.getTime() - d.getTime();
  const minutes = Math.round(diffMs / 60000);
  if (minutes < 1) return 'Только что';
  if (minutes < 60) return `${minutes} мин назад`;
  const hours = Math.round(minutes / 60);
  if (hours < 24) return `${hours} ч назад`;
  const days = Math.round(hours / 24);
  if (days === 1) return 'Вчера';
  if (days < 7)   return `${days} дн назад`;
  const sameYear = d.getFullYear() === now.getFullYear();
  return `${d.getDate()} ${RU_MONTHS_GEN[d.getMonth()]}${sameYear ? '' : ' ' + d.getFullYear()}`;
}

function isRecent(iso: string): boolean {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return false;
  return Date.now() - d.getTime() < 24 * 60 * 60 * 1000;
}

const NotificationsScreen = () => {
  const { items, read, unreadCount, loading, loaded, error } = useNotifications();
  const [filter, setFilter] = useState<Filter>('all');

  useEffect(() => {
    void refreshNotifications();
  }, []);

  const filtered = useMemo(
    () => filter === 'unread' ? items.filter((n) => !read.has(n.id)) : items,
    [items, read, filter],
  );

  const groups = useMemo(() => {
    const today: ApiNotificationItem[] = [];
    const earlier: ApiNotificationItem[] = [];
    for (const n of filtered) (isRecent(n.sentAt) ? today : earlier).push(n);
    return { today, earlier };
  }, [filtered]);

  const segItems: SegmentedItem<Filter>[] = [
    { id: 'all',    label: 'Все',           count: items.length },
    { id: 'unread', label: 'Непрочитанные', count: unreadCount },
  ];

  return (
    <>
      <LargeHeader
        title="Уведомления"
        subtitle={unreadCount > 0 ? `${unreadCount} непрочитанных` : 'Всё прочитано'}
        trailing={
          unreadCount > 0
            ? (
              <button
                type="button"
                onClick={() => { void markAllRead(); }}
                style={{
                  padding: '6px 12px', borderRadius: 999,
                  background: T.subtle, border: 'none', cursor: 'pointer',
                  fontFamily: SANS, fontSize: 13, fontWeight: 500, color: T.ink,
                }}
              >Прочитать всё</button>
            )
            : undefined
        }
      />

      <Segmented value={filter} onChange={setFilter} items={segItems} />

      {loading && !loaded ? (
        <Placeholder text="Загрузка…" />
      ) : error ? (
        <ErrorState message={error} onRetry={() => void refreshNotifications({ force: true })} />
      ) : filtered.length === 0 ? (
        <EmptyState filter={filter} />
      ) : (
        <>
          {groups.today.length > 0 && (
            <NotifGroup label="Сегодня" items={groups.today} read={read} />
          )}
          {groups.earlier.length > 0 && (
            <NotifGroup label="Ранее" items={groups.earlier} read={read} />
          )}
        </>
      )}
    </>
  );
};

interface NotifGroupProps {
  label: string;
  items: ApiNotificationItem[];
  read: Set<string>;
}

function NotifGroup({ label, items, read }: NotifGroupProps) {
  return (
    <div style={{ marginBottom: 18 }}>
      <div style={{
        padding: '4px 20px 8px',
        fontSize: 12.5, color: T.ink3, fontWeight: 600,
        letterSpacing: 0.3, textTransform: 'uppercase',
      }}>{label}</div>
      <div style={{ padding: '0 16px', display: 'grid', gap: 8 }}>
        {items.map((n) => (
          <NotifCard key={n.id} n={n} unread={!read.has(n.id)} />
        ))}
      </div>
    </div>
  );
}

interface NotifMeta {
  Icon: LucideIcon;
  fill: string;
  ink: string;
}

function metaFor(kind: Kind): NotifMeta {
  switch (kind) {
    case 'reminder':  return { Icon: Bell,          fill: T.amberFill, ink: T.amberDp };
    case 'schedule':  return { Icon: CalendarClock, fill: T.infoFill,  ink: T.info };
    case 'analytics': return { Icon: BarChart2,     fill: T.okFill,    ink: T.ok };
  }
}

interface NotifCardProps {
  n: ApiNotificationItem;
  unread: boolean;
}

function NotifCard({ n, unread }: NotifCardProps) {
  const meta = metaFor(kindOf(n.type));
  const Icon = meta.Icon;
  return (
    <button
      type="button"
      onClick={() => { if (unread) void markRead(n.id); }}
      style={{
        width: '100%', textAlign: 'left',
        background: T.surface,
        border: `0.5px solid ${T.hairline}`,
        borderRadius: 14,
        padding: 14,
        cursor: unread ? 'pointer' : 'default',
        display: 'flex', gap: 12,
        fontFamily: SANS,
        position: 'relative',
      }}
    >
      <div style={{
        width: 38, height: 38, borderRadius: 11,
        background: meta.fill, color: meta.ink,
        display: 'flex', alignItems: 'center', justifyContent: 'center',
        flexShrink: 0,
      }}>
        <Icon size={18} strokeWidth={1.8} />
      </div>
      <div style={{ flex: 1, minWidth: 0 }}>
        <div style={{
          display: 'flex', alignItems: 'baseline', gap: 8, marginBottom: 3,
        }}>
          <div style={{
            fontSize: 14.5, fontWeight: 600, color: T.ink, lineHeight: 1.25,
            wordBreak: 'break-word',
          }}>{n.title}</div>
          {unread && (
            <span style={{
              width: 8, height: 8, borderRadius: '50%', background: T.amberDp,
              flexShrink: 0, marginTop: 5,
            }} />
          )}
        </div>
        <div style={{
          fontSize: 13, color: T.ink2, lineHeight: 1.45, marginBottom: 6,
          wordBreak: 'break-word',
        }}>{n.text}</div>
        <div style={{ fontSize: 11.5, color: T.ink4 }}>{formatRelative(n.sentAt)}</div>
      </div>
    </button>
  );
}

function EmptyState({ filter }: { filter: Filter }) {
  return (
    <div style={{ padding: '64px 32px 16px', textAlign: 'center' }}>
      <div style={{
        width: 56, height: 56, margin: '0 auto 14px',
        borderRadius: '50%', background: T.amberFill,
        display: 'flex', alignItems: 'center', justifyContent: 'center',
        color: T.amberDp,
      }}>
        <BellOff size={26} strokeWidth={1.6} />
      </div>
      <div style={{
        fontFamily: SERIF, fontSize: 22, color: T.ink, marginBottom: 6,
        letterSpacing: -0.2,
      }}>
        {filter === 'unread' ? 'Всё прочитано' : 'Тишина'}
      </div>
      <div style={{ fontSize: 13.5, color: T.ink3, lineHeight: 1.45 }}>
        Новые уведомления появятся здесь
      </div>
    </div>
  );
}

function Placeholder({ text }: { text: string }) {
  return (
    <div style={{
      padding: '64px 16px', textAlign: 'center',
      color: T.ink3, fontSize: 14,
    }}>{text}</div>
  );
}

export default NotificationsScreen;
