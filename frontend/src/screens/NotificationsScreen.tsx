import { Bell } from 'lucide-react';
import NotificationList from '../components/NotificationList';
import WeeklyDigest from '../components/WeeklyDigest';

export interface Notification {
  id: string;
  type: 'reminder' | 'schedule' | 'analytics';
  title: string;
  message: string;
  timestamp: string;
  read: boolean;
}

const NotificationsScreen = () => {
  const mockNotifications: Notification[] = [
    {
      id: '1',
      type: 'reminder',
      title: 'Напоминание о задаче',
      message: 'Через 30 минут начинается встреча с клиентом',
      timestamp: '2025-11-16T08:30:00',
      read: false,
    },
    {
      id: '2',
      type: 'schedule',
      title: 'Изменение в расписании',
      message: 'Тренировка перенесена на 18:00',
      timestamp: '2025-11-16T07:15:00',
      read: false,
    },
    {
      id: '3',
      type: 'analytics',
      title: 'Недельная аналитика',
      message: 'Вы выполнили 85% запланированных задач',
      timestamp: '2025-11-15T20:00:00',
      read: true,
    },
    {
      id: '4',
      type: 'reminder',
      title: 'Дедлайн приближается',
      message: 'Осталось 2 дня до дедлайна "Подготовить презентацию"',
      timestamp: '2025-11-15T10:00:00',
      read: true,
    },
  ];

  const unreadCount = mockNotifications.filter((n) => !n.read).length;

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
            <NotificationList notifications={mockNotifications} />
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
