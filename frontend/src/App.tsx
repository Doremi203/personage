import { useState } from 'react';
import TasksScreen from './screens/TasksScreen';
import ScheduleScreen from './screens/ScheduleScreen';
import NotificationsScreen from './screens/NotificationsScreen';
import SettingsScreen from './screens/SettingsScreen';
import AuthScreen from './screens/AuthScreen';
import ResetPasswordScreen from './screens/ResetPasswordScreen';
import Sidebar from './components/Sidebar';
import PWAInstallPrompt from './components/PWAInstallPrompt';
import WelcomeScreen from './components/WelcomeScreen';
import { isAuthenticated, logout } from './utils/authService';

const ONBOARDING_KEY = 'personage_onboarding_completed';

type Screen = 'tasks' | 'schedule' | 'notifications' | 'settings';

function getResetToken(): string | null {
  return new URLSearchParams(window.location.search).get('token');
}

function clearResetToken(): void {
  const url = new URL(window.location.href);
  url.searchParams.delete('token');
  window.history.replaceState({}, '', url.pathname + url.search);
}

function App() {
  const [resetToken] = useState<string | null>(() => getResetToken());
  const [authenticated, setAuthenticated] = useState(() => isAuthenticated());
  const [currentScreen, setCurrentScreen] = useState<Screen>('tasks');
  const [onboardingComplete, setOnboardingComplete] = useState(
    () => localStorage.getItem(ONBOARDING_KEY) === 'true',
  );

  const handleAuthSuccess = () => {
    setAuthenticated(true);
  };

  const handleResetSuccess = () => {
    clearResetToken();
    setAuthenticated(true);
  };

  const handleLogout = () => {
    logout();
    setAuthenticated(false);
  };

  const handleOnboardingComplete = () => {
    localStorage.setItem(ONBOARDING_KEY, 'true');
    setOnboardingComplete(true);
  };

  if (resetToken) {
    return <ResetPasswordScreen token={resetToken} onSuccess={handleResetSuccess} />;
  }

  if (!authenticated) {
    return <AuthScreen onAuthSuccess={handleAuthSuccess} />;
  }

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
      <Sidebar
        currentScreen={currentScreen}
        onScreenChange={setCurrentScreen}
        onLogout={handleLogout}
      />
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
