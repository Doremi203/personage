import {useState} from 'react';
import {Bell, BellOff} from 'lucide-react';
import {isPushSupported, type PushSetupResult, setupPushNotifications,} from '../utils/pushNotifications';

interface WelcomeScreenProps {
    onComplete: () => void;
}

type ScreenState =
    | 'welcome'
    | 'loading'
    | 'success'
    | 'denied'
    | 'unsupported'
    | 'error';

const WelcomeScreen = ({onComplete}: WelcomeScreenProps) => {
    const [state, setState] = useState<ScreenState>('welcome');
    const [errorMessage, setErrorMessage] = useState<string>('');

    const handleEnableNotifications = async () => {
        setState('loading');

        const result: PushSetupResult = await setupPushNotifications();

        switch (result.status) {
            case 'subscribed':
                setState('success');
                // Auto-dismiss after a short delay
                setTimeout(onComplete, 1500);
                break;
            case 'denied':
                setState('denied');
                break;
            case 'unsupported':
                setState('unsupported');
                break;
            case 'error':
                setErrorMessage(result.error.message);
                setState('error');
                break;
        }
    };

    const handleSkip = () => {
        onComplete();
    };

    const showUnsupportedHint = !isPushSupported()

    return (
        <div className="fixed inset-0 z-[100] flex items-center justify-center bg-[#2D2F31]/60 backdrop-blur-sm p-4">
            <div className="w-full max-w-md bg-white rounded-3xl shadow-2xl overflow-hidden animate-slide-up">
                {/* Header with gradient */}
                <div className="bg-gradient-to-br from-[#5C6BFF] to-[#7C8CFF] px-8 pt-10 pb-8 text-center">
                    <div className="w-20 h-20 rounded-2xl overflow-hidden flex items-center justify-center mx-auto mb-5">
                        <img src="/icon-192x192.png" alt="Personage" className="w-full h-full object-contain"/>
                    </div>
                    <h1 className="text-2xl font-bold text-white mb-2">
                        Добро пожаловать в Personage!
                    </h1>
                    <p className="text-white/80 text-sm leading-relaxed">
                        Ваш персональный ассистент для управления задачами и расписанием
                    </p>
                </div>

                {/* Content area */}
                <div className="px-8 py-6">
                    {state === 'welcome' && (
                        <>
                            {/* Notification explanation */}
                            <div className="flex items-start gap-4 mb-6 p-4 bg-[#F7F8FA] rounded-xl">
                                <div
                                    className="w-10 h-10 bg-[#FF8A65]/10 rounded-xl flex items-center justify-center flex-shrink-0">
                                    <Bell size={20} className="text-[#FF8A65]"/>
                                </div>
                                <div>
                                    <h3 className="font-semibold text-[#2D2F31] text-sm mb-1">
                                        Уведомления
                                    </h3>
                                    <p className="text-xs text-gray-500 leading-relaxed">
                                        Получайте напоминания о задачах, изменениях в расписании и
                                        еженедельную аналитику продуктивности
                                    </p>
                                </div>
                            </div>

                            {showUnsupportedHint && (
                                <div className="mb-4 p-3 bg-gray-50 border border-gray-200 rounded-xl">
                                    <p className="text-xs text-gray-600 leading-relaxed">
                                        Ваш браузер не поддерживает push-уведомления. Вы можете
                                        продолжить использовать приложение без них.
                                    </p>
                                </div>
                            )}

                            {/* Action buttons */}
                            <div className="space-y-3">
                                {showUnsupportedHint ? null : (
                                    <button
                                        onClick={handleEnableNotifications}
                                        className="w-full px-4 py-3 bg-[#5C6BFF] text-white rounded-xl hover:bg-[#4C5BEF] transition-colors font-medium shadow-lg shadow-[#5C6BFF]/20 text-sm"
                                    >
                                        Включить уведомления
                                    </button>
                                )}

                                <button
                                    onClick={handleSkip}
                                    className="w-full px-4 py-3 text-gray-500 hover:text-gray-700 transition-colors font-medium text-sm"
                                >
                                    Пропустить
                                </button>
                            </div>
                        </>
                    )}

                    {state === 'loading' && (
                        <div className="text-center py-8">
                            <div
                                className="w-12 h-12 border-4 border-[#5C6BFF]/20 border-t-[#5C6BFF] rounded-full animate-spin mx-auto mb-4"/>
                            <p className="text-sm text-gray-600">
                                Настраиваем уведомления...
                            </p>
                        </div>
                    )}

                    {state === 'success' && (
                        <div className="text-center py-8">
                            <div
                                className="w-16 h-16 bg-[#4CB782]/10 rounded-full flex items-center justify-center mx-auto mb-4">
                                <Bell size={28} className="text-[#4CB782]"/>
                            </div>
                            <h3 className="font-semibold text-[#2D2F31] mb-1">
                                Уведомления включены!
                            </h3>
                            <p className="text-sm text-gray-500">
                                Вы будете получать важные обновления
                            </p>
                        </div>
                    )}

                    {state === 'denied' && (
                        <div className="text-center py-6">
                            <div
                                className="w-16 h-16 bg-gray-100 rounded-full flex items-center justify-center mx-auto mb-4">
                                <BellOff size={28} className="text-gray-400"/>
                            </div>
                            <h3 className="font-semibold text-[#2D2F31] mb-1">
                                Уведомления отключены
                            </h3>
                            <p className="text-sm text-gray-500 mb-6">
                                Вы можете включить их позже в настройках браузера
                            </p>
                            <button
                                onClick={handleSkip}
                                className="w-full px-4 py-3 bg-[#5C6BFF] text-white rounded-xl hover:bg-[#4C5BEF] transition-colors font-medium shadow-lg shadow-[#5C6BFF]/20 text-sm"
                            >
                                Продолжить
                            </button>
                        </div>
                    )}

                    {state === 'unsupported' && (
                        <div className="text-center py-6">
                            <div
                                className="w-16 h-16 bg-gray-100 rounded-full flex items-center justify-center mx-auto mb-4">
                                <BellOff size={28} className="text-gray-400"/>
                            </div>
                            <h3 className="font-semibold text-[#2D2F31] mb-1">
                                Уведомления недоступны
                            </h3>
                            <p className="text-sm text-gray-500 mb-6">
                                Ваш браузер не поддерживает push-уведомления
                            </p>
                            <button
                                onClick={handleSkip}
                                className="w-full px-4 py-3 bg-[#5C6BFF] text-white rounded-xl hover:bg-[#4C5BEF] transition-colors font-medium shadow-lg shadow-[#5C6BFF]/20 text-sm"
                            >
                                Продолжить
                            </button>
                        </div>
                    )}

                    {state === 'error' && (
                        <div className="text-center py-6">
                            <div
                                className="w-16 h-16 bg-red-50 rounded-full flex items-center justify-center mx-auto mb-4">
                                <BellOff size={28} className="text-red-400"/>
                            </div>
                            <h3 className="font-semibold text-[#2D2F31] mb-1">
                                Произошла ошибка
                            </h3>
                            <p className="text-sm text-gray-500 mb-2">
                                Не удалось настроить уведомления
                            </p>
                            {errorMessage && (
                                <p className="text-xs text-red-400 mb-4 font-mono bg-red-50 p-2 rounded-lg">
                                    {errorMessage}
                                </p>
                            )}
                            <div className="space-y-3">
                                <button
                                    onClick={handleEnableNotifications}
                                    className="w-full px-4 py-3 bg-[#5C6BFF] text-white rounded-xl hover:bg-[#4C5BEF] transition-colors font-medium shadow-lg shadow-[#5C6BFF]/20 text-sm"
                                >
                                    Попробовать снова
                                </button>
                                <button
                                    onClick={handleSkip}
                                    className="w-full px-4 py-3 text-gray-500 hover:text-gray-700 transition-colors font-medium text-sm"
                                >
                                    Пропустить
                                </button>
                            </div>
                        </div>
                    )}
                </div>
            </div>
        </div>
    );
};

export default WelcomeScreen;
