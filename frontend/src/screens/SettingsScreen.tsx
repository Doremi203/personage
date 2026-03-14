import { useState } from 'react';
import { Clock, Bell, Folder, User, Globe, Mail, CheckCircle, Loader2 } from 'lucide-react';
import {
  getConnectedGmailEmail,
  clearConnectedGmailEmail,
  startGmailAuth,
  getUserInfo,
} from '../utils/authService';

const SettingsScreen = () => {
  const categories = ['Работа', 'Учёба', 'Личное', 'Финансы', 'Здоровье'];
  const [connectedGmailEmail, setConnectedGmailEmailState] = useState<string | null>(
    () => getConnectedGmailEmail(),
  );

  const [gmailLoading, setGmailLoading] = useState(false);
  const [gmailError, setGmailError] = useState<string | null>(null);

  const handleConnectGmail = async () => {
    setGmailError(null);
    setGmailLoading(true);
    try {
      const userInfo = getUserInfo();
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
                <select className="w-full px-4 py-2.5 bg-[#F7F8FA] border border-gray-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-[#5C6BFF] focus:border-transparent">
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
              <div className="w-10 h-10 bg-[#4CB782]/10 rounded-xl flex items-center justify-center flex-shrink-0">
                <Clock size={20} className="text-[#4CB782]" />
              </div>
              <div>
                <h3 className="font-semibold text-base md:text-lg text-[#2D2F31]">Рабочие часы</h3>
                <p className="text-xs md:text-sm text-gray-500">Установите ваше рабочее время</p>
              </div>
            </div>

            <div className="grid grid-cols-2 gap-4 md:gap-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Начало рабочего дня
                </label>
                <input
                  type="time"
                  defaultValue="09:00"
                  className="w-full px-4 py-2.5 bg-[#F7F8FA] border border-gray-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-[#5C6BFF] focus:border-transparent"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Конец рабочего дня
                </label>
                <input
                  type="time"
                  defaultValue="18:00"
                  className="w-full px-4 py-2.5 bg-[#F7F8FA] border border-gray-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-[#5C6BFF] focus:border-transparent"
                />
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
              <div className="flex items-center justify-between p-4 bg-[#F7F8FA] rounded-xl">
                <div>
                  <p className="font-medium text-[#2D2F31]">Ежедневный дайджест</p>
                  <p className="text-sm text-gray-500">Получать утренний обзор задач</p>
                </div>
                <label className="relative inline-flex items-center cursor-pointer">
                  <input type="checkbox" defaultChecked className="sr-only peer" />
                  <div className="w-11 h-6 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-[#5C6BFF]/20 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-[#5C6BFF]"></div>
                </label>
              </div>

              <div className="flex items-center justify-between p-4 bg-[#F7F8FA] rounded-xl">
                <div>
                  <p className="font-medium text-[#2D2F31]">Еженедельный дайджест</p>
                  <p className="text-sm text-gray-500">Обзор продуктивности за неделю</p>
                </div>
                <label className="relative inline-flex items-center cursor-pointer">
                  <input type="checkbox" defaultChecked className="sr-only peer" />
                  <div className="w-11 h-6 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-[#5C6BFF]/20 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-[#5C6BFF]"></div>
                </label>
              </div>

              <div className="flex items-center justify-between p-4 bg-[#F7F8FA] rounded-xl">
                <div>
                  <p className="font-medium text-[#2D2F31]">Напоминания о задачах</p>
                  <p className="text-sm text-gray-500">Уведомления о приближающихся дедлайнах</p>
                </div>
                <label className="relative inline-flex items-center cursor-pointer">
                  <input type="checkbox" defaultChecked className="sr-only peer" />
                  <div className="w-11 h-6 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-[#5C6BFF]/20 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-[#5C6BFF]"></div>
                </label>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Тихие часы
                </label>
                <div className="grid grid-cols-2 gap-4">
                  <input
                    type="time"
                    defaultValue="22:00"
                    className="px-4 py-2.5 bg-[#F7F8FA] border border-gray-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-[#5C6BFF] focus:border-transparent"
                  />
                  <input
                    type="time"
                    defaultValue="08:00"
                    className="px-4 py-2.5 bg-[#F7F8FA] border border-gray-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-[#5C6BFF] focus:border-transparent"
                  />
                </div>
                <p className="text-xs text-gray-500 mt-2">Не беспокоить в это время</p>
              </div>
            </div>
          </div>

          <div className="bg-white rounded-2xl border border-gray-200 p-4 md:p-6">
            <div className="flex items-center gap-3 mb-4 md:mb-6">
              <div className="w-10 h-10 bg-gray-100 rounded-xl flex items-center justify-center flex-shrink-0">
                <Folder size={20} className="text-gray-600" />
              </div>
              <div>
                <h3 className="font-semibold text-base md:text-lg text-[#2D2F31]">Категории задач</h3>
                <p className="text-xs md:text-sm text-gray-500">Управление категориями</p>
              </div>
            </div>

            <div className="space-y-2 mb-4">
              {categories.map((category, index) => (
                <div
                  key={index}
                  className="flex items-center justify-between p-3 bg-[#F7F8FA] rounded-xl"
                >
                  <span className="font-medium text-[#2D2F31]">{category}</span>
                  <button className="text-sm text-[#FF8A65] hover:text-[#FF7A55] font-medium">
                    Удалить
                  </button>
                </div>
              ))}
            </div>

            <button className="w-full py-2.5 border-2 border-dashed border-gray-300 text-gray-600 rounded-xl hover:border-[#5C6BFF] hover:text-[#5C6BFF] transition-colors font-medium">
              + Добавить категорию
            </button>
          </div>

          <div className="bg-white rounded-2xl border border-gray-200 p-4 md:p-6">
            <div className="flex items-center gap-3 mb-4 md:mb-6">
              <div className="w-10 h-10 bg-[#EA4335]/10 rounded-xl flex items-center justify-center flex-shrink-0">
                <Mail size={20} className="text-[#EA4335]" />
              </div>
              <div>
                <h3 className="font-semibold text-base md:text-lg text-[#2D2F31]">Gmail</h3>
                <p className="text-xs md:text-sm text-gray-500">Подключите Gmail для чтения данных из почты</p>
              </div>
            </div>

            {gmailError && (
              <div className="mb-4 p-3 rounded-lg bg-red-50 border border-red-100 text-sm text-red-600">
                {gmailError}
              </div>
            )}

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
                  defaultValue="Пользователь"
                  className="w-full px-4 py-2.5 bg-[#F7F8FA] border border-gray-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-[#5C6BFF] focus:border-transparent"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Email
                </label>
                <input
                  type="email"
                  defaultValue="user@email.com"
                  className="w-full px-4 py-2.5 bg-[#F7F8FA] border border-gray-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-[#5C6BFF] focus:border-transparent"
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
