import { useState } from 'react';
import TasksScreen from './screens/TasksScreen';
import ScheduleScreen from './screens/ScheduleScreen';
import NotificationsScreen from './screens/NotificationsScreen';
import SettingsScreen from './screens/SettingsScreen';
import Sidebar from './components/Sidebar';
import PWAInstallPrompt from './components/PWAInstallPrompt';
import WelcomeScreen from './components/WelcomeScreen';

const ONBOARDING_KEY = 'personage_onboarding_completed';

type Screen = 'tasks' | 'schedule' | 'notifications' | 'settings';

function App() {
  const [currentScreen, setCurrentScreen] = useState<Screen>('tasks');
  const [onboardingComplete, setOnboardingComplete] = useState(
    () => localStorage.getItem(ONBOARDING_KEY) === 'true',
  );

  const handleOnboardingComplete = () => {
    localStorage.setItem(ONBOARDING_KEY, 'true');
    setOnboardingComplete(true);
  };

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
      {!onboardingComplete && (
        <WelcomeScreen onComplete={handleOnboardingComplete} />
      )}
    </div>
  );
}

export default App;
