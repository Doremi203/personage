import { CheckSquare, Calendar, Bell, Settings, Sparkles, Menu, X, LogOut } from 'lucide-react';
import { useState } from 'react';
import { getUserInfo } from '../utils/authService';

type Screen = 'tasks' | 'schedule' | 'notifications' | 'settings';

interface SidebarProps {
  currentScreen: Screen;
  onScreenChange: (screen: Screen) => void;
  onLogout: () => void;
}

const Sidebar = ({ currentScreen, onScreenChange, onLogout }: SidebarProps) => {
  const [isMobileOpen, setIsMobileOpen] = useState(false);
  const userInfo = getUserInfo();

  const menuItems = [
    { id: 'tasks' as Screen, icon: CheckSquare, label: 'Задачи' },
    { id: 'schedule' as Screen, icon: Calendar, label: 'Расписание' },
    { id: 'notifications' as Screen, icon: Bell, label: 'Уведомления' },
    { id: 'settings' as Screen, icon: Settings, label: 'Настройки' },
  ];

  const handleNavigation = (screen: Screen) => {
    onScreenChange(screen);
    setIsMobileOpen(false);
  };

  const userInitial = userInfo?.name
    ? userInfo.name.charAt(0).toUpperCase()
    : (userInfo?.email?.charAt(0).toUpperCase() ?? 'П');

  return (
    <>
      <aside className="hidden md:flex md:w-64 bg-[#2D2F31] text-white flex-col">
        <div className="p-6 border-b border-white/10">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 bg-gradient-to-br from-[#5C6BFF] to-[#7C8CFF] rounded-xl flex items-center justify-center">
              <Sparkles size={20} className="text-white" />
            </div>
            <div>
              <h1 className="font-semibold text-lg">Personage</h1>
              <p className="text-xs text-white/60">Personal Assistant</p>
            </div>
          </div>
        </div>

        <nav className="flex-1 p-4">
          {menuItems.map((item) => {
            const Icon = item.icon;
            const isActive = currentScreen === item.id;
            return (
              <button
                key={item.id}
                onClick={() => handleNavigation(item.id)}
                className={`w-full flex items-center gap-3 px-4 py-3 rounded-xl mb-2 transition-all ${
                  isActive
                    ? 'bg-[#5C6BFF] text-white shadow-lg shadow-[#5C6BFF]/20'
                    : 'text-white/70 hover:bg-white/5 hover:text-white'
                }`}
              >
                <Icon size={20} />
                <span className="font-medium">{item.label}</span>
              </button>
            );
          })}
        </nav>

        <div className="p-6 border-t border-white/10">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 bg-gradient-to-br from-[#4CB782] to-[#5CC792] rounded-full flex items-center justify-center text-white font-semibold text-sm flex-shrink-0">
              {userInitial}
            </div>
            <div className="flex-1 min-w-0">
              <p className="font-medium text-sm truncate">
                {userInfo?.name ?? 'Пользователь'}
              </p>
              <p className="text-xs text-white/60 truncate">
                {userInfo?.email ?? ''}
              </p>
            </div>
            <button
              onClick={onLogout}
              title="Выйти"
              className="p-1.5 rounded-lg text-white/50 hover:text-white hover:bg-white/10 transition-colors flex-shrink-0"
            >
              <LogOut size={16} />
            </button>
          </div>
        </div>
      </aside>

      <div className="md:hidden fixed top-0 left-0 right-0 h-16 bg-[#2D2F31] text-white z-50 flex items-center justify-between px-4">
        <div className="flex items-center gap-2">
          <div className="w-8 h-8 bg-gradient-to-br from-[#5C6BFF] to-[#7C8CFF] rounded-lg flex items-center justify-center">
            <Sparkles size={16} className="text-white" />
          </div>
          <h1 className="font-semibold text-base">Personage</h1>
        </div>
        <button
          onClick={() => setIsMobileOpen(!isMobileOpen)}
          className="p-2 hover:bg-white/10 rounded-lg transition-colors"
        >
          {isMobileOpen ? <X size={24} /> : <Menu size={24} />}
        </button>
      </div>

      {isMobileOpen && (
        <div className="md:hidden fixed inset-0 bg-black/50 z-40" onClick={() => setIsMobileOpen(false)} />
      )}

      <nav
        className={`md:hidden fixed left-0 top-16 bottom-0 w-64 bg-[#2D2F31] text-white flex flex-col z-40 transform transition-transform ${
          isMobileOpen ? 'translate-x-0' : '-translate-x-full'
        }`}
      >
        <div className="p-4 flex-1">
          {menuItems.map((item) => {
            const Icon = item.icon;
            const isActive = currentScreen === item.id;
            return (
              <button
                key={item.id}
                onClick={() => handleNavigation(item.id)}
                className={`w-full flex items-center gap-3 px-4 py-3 rounded-xl mb-2 transition-all ${
                  isActive
                    ? 'bg-[#5C6BFF] text-white shadow-lg shadow-[#5C6BFF]/20'
                    : 'text-white/70 hover:bg-white/5 hover:text-white'
                }`}
              >
                <Icon size={20} />
                <span className="font-medium">{item.label}</span>
              </button>
            );
          })}
        </div>

        <div className="p-4 border-t border-white/10">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 bg-gradient-to-br from-[#4CB782] to-[#5CC792] rounded-full flex items-center justify-center text-white font-semibold text-sm flex-shrink-0">
              {userInitial}
            </div>
            <div className="flex-1 min-w-0">
              <p className="font-medium text-sm truncate">
                {userInfo?.name ?? 'Пользователь'}
              </p>
              <p className="text-xs text-white/60 truncate">
                {userInfo?.email ?? ''}
              </p>
            </div>
            <button
              onClick={onLogout}
              title="Выйти"
              className="p-1.5 rounded-lg text-white/50 hover:text-white hover:bg-white/10 transition-colors flex-shrink-0"
            >
              <LogOut size={16} />
            </button>
          </div>
        </div>
      </nav>
    </>
  );
};

export default Sidebar;
