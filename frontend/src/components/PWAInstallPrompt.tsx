import { useState, useEffect } from 'react';
import { Download, X, Share, PlusSquare } from 'lucide-react';
import { isIos, isStandalonePWA } from '../utils/pushNotifications';

const DISMISSED_KEY = 'personage_pwa_prompt_dismissed';

const PWAInstallPrompt = () => {
  const isDismissed = localStorage.getItem(DISMISSED_KEY) === 'true';
  const isStandalone = isStandalonePWA();
  const shouldShowIOSGuide = !isDismissed && !isStandalone && isIos();
  const shouldListenForInstall = !isDismissed && !isStandalone && !isIos();

  const [deferredPrompt, setDeferredPrompt] = useState<Event | null>(null);
  const [showPrompt, setShowPrompt] = useState(false);
  const [showIOSGuide, setShowIOSGuide] = useState(shouldShowIOSGuide);

  useEffect(() => {
    if (!shouldListenForInstall) return;

    const handler = (e: Event) => {
      e.preventDefault();
      setDeferredPrompt(e);
      setShowPrompt(true);
    };

    window.addEventListener('beforeinstallprompt', handler);

    return () => {
      window.removeEventListener('beforeinstallprompt', handler);
    };
  }, [shouldListenForInstall]);

  const handleInstall = async () => {
    if (!deferredPrompt) return;

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const promptEvent = deferredPrompt as any;
    promptEvent.prompt();
    const { outcome } = await promptEvent.userChoice;

    if (outcome === 'accepted') {
      console.log('PWA installation accepted');
    }

    setDeferredPrompt(null);
    setShowPrompt(false);
  };

  const handleDismiss = () => {
    localStorage.setItem(DISMISSED_KEY, 'true');
    setShowPrompt(false);
    setShowIOSGuide(false);
  };

  // Chromium install prompt
  if (showPrompt) {
    return (
      <div className="fixed bottom-4 left-4 right-4 md:left-auto md:right-4 md:w-96 bg-white rounded-2xl shadow-2xl border border-gray-200 p-4 z-50 animate-slide-up">
        <button
          onClick={handleDismiss}
          className="absolute top-3 right-3 p-1 hover:bg-gray-100 rounded-lg transition-colors"
        >
          <X size={18} className="text-gray-500" />
        </button>

        <div className="flex items-start gap-4">
          <div className="w-12 h-12 bg-gradient-to-br from-[#5C6BFF] to-[#7C8CFF] rounded-xl flex items-center justify-center flex-shrink-0">
            <Download size={24} className="text-white" />
          </div>
          <div className="flex-1">
            <h3 className="font-semibold text-[#2D2F31] mb-1">
              Установить Personage
            </h3>
            <p className="text-sm text-gray-600 mb-3">
              Установите приложение для быстрого доступа и работы офлайн
            </p>
            <button
              onClick={handleInstall}
              className="w-full px-4 py-2.5 bg-[#5C6BFF] text-white rounded-xl hover:bg-[#4C5BEF] transition-colors font-medium shadow-lg shadow-[#5C6BFF]/20"
            >
              Установить
            </button>
          </div>
        </div>
      </div>
    );
  }

  if (showIOSGuide) {
    return (
      <div className="fixed bottom-4 left-4 right-4 md:left-auto md:right-4 md:w-96 bg-white rounded-2xl shadow-2xl border border-gray-200 p-4 z-50 animate-slide-up">
        <button
          onClick={handleDismiss}
          className="absolute top-3 right-3 p-1 hover:bg-gray-100 rounded-lg transition-colors"
        >
          <X size={18} className="text-gray-500" />
        </button>

        <div className="flex items-start gap-4">
          <div className="w-12 h-12 bg-gradient-to-br from-[#5C6BFF] to-[#7C8CFF] rounded-xl flex items-center justify-center flex-shrink-0">
            <Download size={24} className="text-white" />
          </div>
          <div className="flex-1">
            <h3 className="font-semibold text-[#2D2F31] mb-2">
              Установить Personage
            </h3>
            <div className="space-y-2 mb-1">
              <p className="text-sm text-gray-600">
                <span className="inline-flex items-center gap-1">
                  1. Нажмите
                  <Share size={14} className="text-[#5C6BFF]" />
                  <span className="font-medium">«Поделиться»</span>
                </span>
              </p>
              <p className="text-sm text-gray-600">
                <span className="inline-flex items-center gap-1">
                  2. Выберите
                  <PlusSquare size={14} className="text-[#5C6BFF]" />
                  <span className="font-medium">«На экран Домой»</span>
                </span>
              </p>
            </div>
          </div>
        </div>
      </div>
    );
  }

  return null;
};

export default PWAInstallPrompt;
