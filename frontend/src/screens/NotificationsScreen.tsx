import { useState, useEffect } from 'react';
import { Bell } from 'lucide-react';
import NotificationList from '../components/NotificationList';
import WeeklyDigest from '../components/WeeklyDigest';
import { listNotifications, type ApiNotificationItem } from '../utils/notificatorService';

export interface Notification {
  id: string;
  type: 'reminder' | 'schedule' | 'analytics';
  title: string;
  message: string;
  timestamp: string;
  read: boolean;
}

function apiTypeToLocal(type: string): Notification['type'] {
  if (type === 'schedule_change') return 'schedule';
  if (type === 'analytics') return 'analytics';
  return 'reminder';
}

function apiItemToNotification(item: ApiNotificationItem): Notification {
  return {
    id: item.id,
    type: apiTypeToLocal(item.type),
    title: item.title,
    message: item.text,
    timestamp: item.sentAt,
    read: true,
  };
}

const PAGE_SIZE = 10;

const NotificationsScreen = () => {
  const [notifications, setNotifications] = useState<Notification[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    void (async () => {
      try {
        const data = await listNotifications(1, PAGE_SIZE);
        if (!cancelled) {
          setNotifications((data.notifications ?? []).map(apiItemToNotification));
        }
      } catch (err) {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : 'Не удалось загрузить уведомления');
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  const unreadCount = notifications.filter((n) => !n.read).length;

  return (
    <div className="h-full flex flex-col lg:flex-row md:pt-0 pt-16">
      <div className="flex-1 overflow-auto">
        <div className="p-4 md:p-8">
          <div className="mb-6 md:mb-8">
            <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-2 mb-2">
              <h2 className="text-xl md:text-2xl font-bold text-[#2D2F31]">Уведомления</h2>
              {unreadCount > 0 && (
                <span className="px-3 py-1 bg-[#FF8A65] text-white text-xs md:text-sm font-medium rounded-full w-fit">
                  {unreadCount} новых
                </span>
              )}
            </div>
            <p className="text-xs md:text-base text-gray-500">Все важные обновления в одном месте</p>
          </div>

          <div className="mb-8">
            <div className="flex items-center gap-2 mb-4">
              <Bell size={18} className="text-[#5C6BFF]" />
              <h3 className="font-semibold text-base md:text-lg text-[#2D2F31]">Последние уведомления</h3>
            </div>
            {loading ? (
              <div className="text-sm text-gray-500 py-4">Загрузка...</div>
            ) : error ? (
              <div className="text-sm text-red-500 py-4">{error}</div>
            ) : notifications.length === 0 ? (
              <div className="text-sm text-gray-500 py-4">Уведомлений пока нет</div>
            ) : (
              <NotificationList notifications={notifications} />
            )}
          </div>
        </div>
      </div>

      <div className="hidden lg:block lg:w-[500px] bg-white border-l border-gray-200 overflow-auto">
        <WeeklyDigest />
      </div>
    </div>
  );
};

export default NotificationsScreen;
