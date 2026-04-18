import { useState, useEffect } from 'react';
import { Bell, User, Globe, Mail, CheckCircle, Loader2, MessageCircle } from 'lucide-react';
import {
  getConnectedGmailEmail,
  clearConnectedGmailEmail,
  startGmailAuth,
  getUserInfo,
  fetchCurrentUser,
  setConnectedGmailEmail,
  type UserApiResponse,
} from '../utils/authService';
import { toggleNotification, getNotificationSettings } from '../utils/notificatorService';

const TYPE_LABELS: Record<string, { title: string; description: string }> = {
  upcoming_event: {
    title: 'Напоминания о задачах',
    description: 'Уведомления о приближающихся дедлайнах',
  },
  schedule_change: {
    title: 'Изменения в расписании',
    description: 'Уведомления об изменениях в расписании',
  },
};

interface NotificationSettingState {
  type: string;
  enabled: boolean;
  toggling: boolean;
}

const SettingsScreen = () => {
  const [connectedGmailEmail, setConnectedGmailEmailState] = useState<string | null>(
    () => getConnectedGmailEmail(),
  );

  const [gmailLoading, setGmailLoading] = useState(false);
  const [gmailError, setGmailError] = useState<string | null>(null);

  const [notificationSettings, setNotificationSettings] = useState<NotificationSettingState[]>([]);

  const [userData, setUserData] = useState<UserApiResponse | null>(null);
  const [userLoading, setUserLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
    void (async () => {
      try {
        const { settings } = await getNotificationSettings();
        if (!cancelled) {
          setNotificationSettings(settings.map((s) => ({ ...s, toggling: false })));
        }
      } catch (err) {
        console.error('Failed to fetch notification settings:', err);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    let cancelled = false;
    void (async () => {
      try {
        const data = await fetchCurrentUser();
        if (!cancelled) {
          setUserData(data);
          if (data.gmailIntegration.enabled && data.gmailIntegration.gmail) {
            setConnectedGmailEmail(data.gmailIntegration.gmail);
            setConnectedGmailEmailState(data.gmailIntegration.gmail);
          }
        }
      } catch (err) {
        console.error('Failed to fetch user data:', err);
      } finally {
        if (!cancelled) setUserLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  const handleConnectGmail = async () => {
    setGmailError(null);
    setGmailLoading(true);
    try {
      const userInfo = userData ?? getUserInfo();
      if (!userInfo?.email) {
        setGmailError('Не удалось определить email пользователя');
        return;
      }
      const { authorizationUrl } = await startGmailAuth(
        userInfo.email,
        window.location.origin,
      );
      window.location.href = authorizationUrl;
    } catch (err) {
      setGmailError(
        err instanceof Error ? err.message : 'Не удалось запустить авторизацию Gmail',
      );
      setGmailLoading(false);
    }
  };

  const handleDisconnectGmail = () => {
    clearConnectedGmailEmail();
    setConnectedGmailEmailState(null);
    setGmailError(null);
  };

  const handleConnectGmailClick = () => {
    void handleConnectGmail();
  };

  const handleToggle = (type: string) => {
    setNotificationSettings((prev) =>
      prev.map((s) => (s.type === type ? { ...s, toggling: true } : s)),
    );
    toggleNotification(type)
      .then((enabled) => {
        setNotificationSettings((prev) =>
          prev.map((s) => (s.type === type ? { ...s, enabled, toggling: false } : s)),
        );
      })
      .catch((err: unknown) => {
        console.error('Failed to toggle notification setting:', err);
        setNotificationSettings((prev) =>
          prev.map((s) => (s.type === type ? { ...s, toggling: false } : s)),
        );
      });
  };

  const cachedUserInfo = getUserInfo();
  const displayName = userLoading ? '' : (userData?.name ?? cachedUserInfo?.name ?? '');
  const displayEmail = userLoading ? '' : (userData?.email ?? cachedUserInfo?.email ?? '');
  const accountPlaceholder = userLoading ? 'Загрузка...' : undefined;

  return (
    <div className="h-full overflow-auto md:pt-0 pt-16">
      <div className="max-w-4xl mx-auto p-4 md:p-8">
        <div className="mb-6 md:mb-8">
          <h2 className="text-xl md:text-2xl font-bold text-[#2D2F31] mb-2">Настройки</h2>
          <p className="text-xs md:text-base text-gray-500">Настройте ваш персональный ассистент</p>
        </div>

        <div className="space-y-4 md:space-y-6">
          <div className="bg-white rounded-2xl border border-gray-200 p-4 md:p-6">
            <div className="flex items-center gap-3 mb-4 md:mb-6">
              <div className="w-10 h-10 bg-[#5C6BFF]/10 rounded-xl flex items-center justify-center flex-shrink-0">
                <Globe size={20} className="text-[#5C6BFF]" />
              </div>
              <div>
                <h3 className="font-semibold text-base md:text-lg text-[#2D2F31]">Региональные настройки</h3>
                <p className="text-xs md:text-sm text-gray-500">Часовой пояс и формат времени</p>
              </div>
            </div>

            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Часовой пояс
                </label>
                <select className="w-full px-4 py-2.5 bg-[#F7F8FA] border border-gray-200 rounded-xl text-base focus:outline-none focus:ring-2 focus:ring-[#5C6BFF] focus:border-transparent">
                  <option>Europe/Moscow (GMT+3)</option>
                  <option>Europe/London (GMT+0)</option>
                  <option>America/New_York (GMT-5)</option>
                  <option>Asia/Tokyo (GMT+9)</option>
                </select>
              </div>
            </div>
          </div>

          <div className="bg-white rounded-2xl border border-gray-200 p-4 md:p-6">
            <div className="flex items-center gap-3 mb-4 md:mb-6">
              <div className="w-10 h-10 bg-[#FF8A65]/10 rounded-xl flex items-center justify-center flex-shrink-0">
                <Bell size={20} className="text-[#FF8A65]" />
              </div>
              <div>
                <h3 className="font-semibold text-base md:text-lg text-[#2D2F31]">Уведомления</h3>
                <p className="text-xs md:text-sm text-gray-500">Управление уведомлениями</p>
              </div>
            </div>

            <div className="space-y-4">
              {notificationSettings.map(({ type, enabled, toggling }) => {
                const label = TYPE_LABELS[type];
                return (
                  <div key={type} className="flex items-center justify-between p-4 bg-[#F7F8FA] rounded-xl">
                    <div>
                      <p className="font-medium text-[#2D2F31]">{label?.title ?? type}</p>
                      {label?.description && (
                        <p className="text-sm text-gray-500">{label.description}</p>
                      )}
                    </div>
                    <label className="relative inline-flex items-center cursor-pointer">
                      <input
                        type="checkbox"
                        checked={enabled}
                        onChange={() => handleToggle(type)}
                        disabled={toggling}
                        className="sr-only peer"
                      />
                      <div className="w-11 h-6 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-[#5C6BFF]/20 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-[#5C6BFF]"></div>
                    </label>
                  </div>
                );
              })}
            </div>
          </div>

          <div className="bg-white rounded-2xl border border-gray-200 p-4 md:p-6">
            <div className="flex items-center gap-3 mb-4 md:mb-6">
              <div className="w-10 h-10 bg-gray-100 rounded-xl flex items-center justify-center flex-shrink-0">
                <Mail size={20} className="text-gray-600" />
              </div>
              <div>
                <h3 className="font-semibold text-base md:text-lg text-[#2D2F31]">Подключённые сервисы</h3>
                <p className="text-xs md:text-sm text-gray-500">Управление интеграциями</p>
              </div>
            </div>

            {gmailError && (
              <div className="mb-4 p-3 rounded-lg bg-red-50 border border-red-100 text-sm text-red-600">
                {gmailError}
              </div>
            )}

            <div className="space-y-3">
              {connectedGmailEmail ? (
                <div className="flex items-center justify-between p-4 bg-[#F7F8FA] rounded-xl">
                  <div className="flex items-center gap-3">
                    <CheckCircle size={18} className="text-[#4CB782] flex-shrink-0" />
                    <div>
                      <p className="font-medium text-[#2D2F31] text-sm">Gmail подключён</p>
                      <p className="text-xs text-gray-500">{connectedGmailEmail}</p>
                    </div>
                  </div>
                  <button
                    onClick={handleDisconnectGmail}
                    className="text-sm text-[#FF8A65] hover:text-[#FF7A55] font-medium flex-shrink-0"
                  >
                    Отключить
                  </button>
                </div>
              ) : (
                <button
                  onClick={handleConnectGmailClick}
                  disabled={gmailLoading}
                  className="w-full flex items-center justify-center gap-2 py-2.5 border border-gray-200 bg-[#F7F8FA] text-[#2D2F31] rounded-xl hover:bg-gray-100 transition-colors font-medium text-sm disabled:opacity-60 disabled:cursor-not-allowed"
                >
                  {gmailLoading ? (
                    <Loader2 size={16} className="animate-spin" />
                  ) : (
                    <Mail size={16} className="text-[#EA4335]" />
                  )}
                  {gmailLoading ? 'Переход к Google...' : 'Подключить Gmail'}
                </button>
              )}

              <div className="flex items-center justify-between p-4 bg-[#F7F8FA] rounded-xl">
                <div className="flex items-center gap-3">
                  {userLoading ? (
                    <Loader2 size={18} className="text-gray-400 animate-spin flex-shrink-0" />
                  ) : userData?.telegramIntegration.enabled ? (
                    <CheckCircle size={18} className="text-[#4CB782] flex-shrink-0" />
                  ) : (
                    <MessageCircle size={18} className="text-[#229ED9] flex-shrink-0" />
                  )}
                  <div>
                    <p className="font-medium text-[#2D2F31] text-sm">Telegram</p>
                    <p className="text-xs text-gray-500">
                      {userLoading
                        ? 'Загрузка...'
                        : userData?.telegramIntegration.enabled
                          ? 'Подключён'
                          : 'Не подключён'}
                    </p>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <div className="bg-white rounded-2xl border border-gray-200 p-4 md:p-6">
            <div className="flex items-center gap-3 mb-4 md:mb-6">
              <div className="w-10 h-10 bg-[#5C6BFF]/10 rounded-xl flex items-center justify-center flex-shrink-0">
                <User size={20} className="text-[#5C6BFF]" />
              </div>
              <div>
                <h3 className="font-semibold text-base md:text-lg text-[#2D2F31]">Информация об аккаунте</h3>
                <p className="text-xs md:text-sm text-gray-500">Ваши личные данные</p>
              </div>
            </div>

            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Имя
                </label>
                <input
                  type="text"
                  value={displayName}
                  readOnly
                  placeholder={accountPlaceholder ?? 'Пользователь'}
                  className="w-full px-4 py-2.5 bg-[#F7F8FA] border border-gray-200 rounded-xl text-base focus:outline-none focus:ring-2 focus:ring-[#5C6BFF] focus:border-transparent"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Email
                </label>
                <input
                  type="email"
                  value={displayEmail}
                  readOnly
                  placeholder={accountPlaceholder ?? 'user@email.com'}
                  className="w-full px-4 py-2.5 bg-[#F7F8FA] border border-gray-200 rounded-xl text-base focus:outline-none focus:ring-2 focus:ring-[#5C6BFF] focus:border-transparent"
                />
              </div>
            </div>
          </div>

          <div className="flex flex-col-reverse md:flex-row justify-end gap-3">
            <button className="px-6 py-2.5 bg-gray-100 text-gray-700 rounded-xl hover:bg-gray-200 transition-colors font-medium text-sm md:text-base">
              Отменить
            </button>
            <button className="px-6 py-2.5 bg-[#5C6BFF] text-white rounded-xl hover:bg-[#4C5BEF] transition-colors font-medium shadow-lg shadow-[#5C6BFF]/20 text-sm md:text-base">
              Сохранить изменения
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};

export default SettingsScreen;
