import { Bell, Calendar, TrendingUp, Check } from 'lucide-react';
import { Notification } from '../screens/NotificationsScreen';

interface NotificationListProps {
  notifications: Notification[];
}

const NotificationList = ({ notifications }: NotificationListProps) => {
  const getIcon = (type: string) => {
    switch (type) {
      case 'reminder':
        return Bell;
      case 'schedule':
        return Calendar;
      case 'analytics':
        return TrendingUp;
      default:
        return Bell;
    }
  };

  const getIconColor = (type: string) => {
    switch (type) {
      case 'reminder':
        return 'bg-[#FF8A65]/10 text-[#FF8A65]';
      case 'schedule':
        return 'bg-[#5C6BFF]/10 text-[#5C6BFF]';
      case 'analytics':
        return 'bg-[#4CB782]/10 text-[#4CB782]';
      default:
        return 'bg-gray-100 text-gray-600';
    }
  };

  const formatTime = (timestamp: string) => {
    const date = new Date(timestamp);
    const now = new Date();
    const diff = now.getTime() - date.getTime();
    const hours = Math.floor(diff / (1000 * 60 * 60));
    const minutes = Math.floor(diff / (1000 * 60));

    if (minutes < 60) {
      return `${minutes} мин назад`;
    } else if (hours < 24) {
      return `${hours} ч назад`;
    } else {
      return date.toLocaleDateString('ru-RU', {
        day: 'numeric',
        month: 'short'
      });
    }
  };

  return (
    <div className="space-y-3">
      {notifications.map((notification) => {
        const Icon = getIcon(notification.type);
        return (
          <div
            key={notification.id}
            className={`bg-white rounded-xl border-2 p-4 transition-all hover:shadow-lg ${
              notification.read
                ? 'border-gray-200'
                : 'border-[#5C6BFF] shadow-md'
            }`}
          >
            <div className="flex items-start gap-4">
              <div className={`w-12 h-12 rounded-xl flex items-center justify-center flex-shrink-0 ${getIconColor(notification.type)}`}>
                <Icon size={20} />
              </div>
              <div className="flex-1 min-w-0">
                <div className="flex items-start justify-between gap-2 mb-1">
                  <h4 className="font-semibold text-[#2D2F31]">{notification.title}</h4>
                  {!notification.read && (
                    <span className="w-2 h-2 bg-[#5C6BFF] rounded-full flex-shrink-0 mt-2" />
                  )}
                </div>
                <p className="text-sm text-gray-600 mb-2">{notification.message}</p>
                <div className="flex items-center justify-between">
                  <span className="text-xs text-gray-500">{formatTime(notification.timestamp)}</span>
                  {!notification.read && (
                    <button className="flex items-center gap-1.5 px-3 py-1.5 bg-[#F7F8FA] hover:bg-gray-200 text-xs font-medium text-gray-700 rounded-lg transition-colors">
                      <Check size={14} />
                      <span>Прочитано</span>
                    </button>
                  )}
                </div>
              </div>
            </div>
          </div>
        );
      })}
    </div>
  );
};

export default NotificationList;
