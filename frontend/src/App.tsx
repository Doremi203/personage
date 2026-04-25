import { useState, useEffect } from 'react';
import TasksScreen from './screens/TasksScreen';
import ScheduleScreen from './screens/ScheduleScreen';
import NotificationsScreen from './screens/NotificationsScreen';
import SettingsScreen from './screens/SettingsScreen';
import AuthScreen from './screens/AuthScreen';
import ResetPasswordScreen from './screens/ResetPasswordScreen';
import { Frame } from './mobile/Frame';
import { Shell, type Tab } from './mobile/Chrome';
import { OnboardingPrompt } from './mobile/OnboardingPrompt';
import { refreshNotifications, useNotifications } from './mobile/notificationsStore';
import {
  AUTH_STATE_CHANGE_EVENT,
  OAUTH_PROVIDER_STORAGE_KEY,
  type OAuthProvider,
  fetchCurrentUser,
  isAuthenticated,
  logout,
  handleGmailCallback,
  handleGoogleCalendarCallback,
} from './utils/authService';
import { clearUserCache } from './utils/userCache';

function getResetToken(): string | null {
  return new URLSearchParams(window.location.search).get('token');
}

function clearResetToken(): void {
  const url = new URL(window.location.href);
  url.searchParams.delete('token');
  window.history.replaceState({}, '', url.pathname + url.search);
}

function getOAuthCallbackParams(): { code: string; state: string } | null {
  const params = new URLSearchParams(window.location.search);
  const code = params.get('code');
  const state = params.get('state');
  if (code && state) return { code, state };
  return null;
}

function clearOAuthCallbackParams(): void {
  const url = new URL(window.location.href);
  url.searchParams.delete('code');
  url.searchParams.delete('state');
  window.history.replaceState({}, '', url.pathname + url.search);
}

function readOAuthProvider(): OAuthProvider {
  const stored = sessionStorage.getItem(OAUTH_PROVIDER_STORAGE_KEY);
  sessionStorage.removeItem(OAUTH_PROVIDER_STORAGE_KEY);
  return stored === 'google-calendar' ? 'google-calendar' : 'gmail';
}

function App() {
  const [resetToken, setResetToken] = useState<string | null>(() => getResetToken());
  const [authenticated, setAuthenticated] = useState(() => isAuthenticated());
  const [currentTab, setCurrentTab] = useState<Tab>('tasks');

  useEffect(() => {
    const syncAuthenticatedState = () => {
      setAuthenticated(isAuthenticated());
    };

    window.addEventListener(AUTH_STATE_CHANGE_EVENT, syncAuthenticatedState);
    return () => {
      window.removeEventListener(AUTH_STATE_CHANGE_EVENT, syncAuthenticatedState);
    };
  }, []);

  useEffect(() => {
    if (authenticated) void refreshNotifications();
  }, [authenticated]);

  useEffect(() => {
    const params = getOAuthCallbackParams();
    if (!params || !isAuthenticated()) return;

    const { code, state } = params;
    const provider = readOAuthProvider();
    void (async () => {
      try {
        const user = await fetchCurrentUser();
        if (provider === 'google-calendar') {
          await handleGoogleCalendarCallback(user.email, code, state, window.location.origin);
        } else {
          await handleGmailCallback(user.email, code, state, window.location.origin);
        }
      } catch (err) {
        console.error(`${provider} callback failed:`, err);
      } finally {
        clearUserCache();
        clearOAuthCallbackParams();
        setCurrentTab('settings');
      }
    })();
  }, []);

  const handleAuthSuccess = () => {
    setAuthenticated(true);
  };

  const handleResetSuccess = () => {
    clearResetToken();
    setResetToken(null);
    setAuthenticated(true);
  };

  const handleLogout = () => {
    logout();
    setAuthenticated(false);
  };

  if (resetToken) {
    return (
      <Frame>
        <ResetPasswordScreen token={resetToken} onSuccess={handleResetSuccess} />
      </Frame>
    );
  }

  if (!authenticated) {
    return (
      <Frame>
        <AuthScreen onAuthSuccess={handleAuthSuccess} />
      </Frame>
    );
  }

  const renderScreen = () => {
    switch (currentTab) {
      case 'tasks':         return <TasksScreen />;
      case 'schedule':      return <ScheduleScreen />;
      case 'notifications': return <NotificationsScreen />;
      case 'settings':      return <SettingsScreen onLogout={handleLogout} />;
    }
  };

  return (
    <Frame>
      <AuthedApp
        currentTab={currentTab}
        onTabChange={setCurrentTab}
        renderScreen={renderScreen}
      />
      <OnboardingPrompt />
    </Frame>
  );
}

interface AuthedAppProps {
  currentTab: Tab;
  onTabChange: (t: Tab) => void;
  renderScreen: () => React.ReactNode;
}

function AuthedApp({ currentTab, onTabChange, renderScreen }: AuthedAppProps) {
  const { unreadCount } = useNotifications();
  return (
    <Shell
      tab={currentTab}
      onTabChange={onTabChange}
      badges={{ notifications: unreadCount }}
    >
      {renderScreen()}
    </Shell>
  );
}

export default App;
