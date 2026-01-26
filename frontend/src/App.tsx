import { useState } from 'react';
import TasksScreen from './screens/TasksScreen';
import ScheduleScreen from './screens/ScheduleScreen';
import NotificationsScreen from './screens/NotificationsScreen';
import SettingsScreen from './screens/SettingsScreen';
import Sidebar from './components/Sidebar';
import PWAInstallPrompt from './components/PWAInstallPrompt';

type Screen = 'tasks' | 'schedule' | 'notifications' | 'settings';

function App() {
  const [currentScreen, setCurrentScreen] = useState<Screen>('tasks');

  const renderScreen = () => {
    switch (currentScreen) {
      case 'tasks':
        return <TasksScreen />;
      case 'schedule':
        return <ScheduleScreen />;
      case 'notifications':
        return <NotificationsScreen />;
      case 'settings':
        return <SettingsScreen />;
      default:
        return <TasksScreen />;
    }
  };

  return (
    <div className="min-h-screen bg-[#F7F8FA] flex">
      <Sidebar currentScreen={currentScreen} onScreenChange={setCurrentScreen} />
      <main className="flex-1 overflow-hidden">
        {renderScreen()}
      </main>
      <PWAInstallPrompt />
    </div>
  );
}

export default App;
