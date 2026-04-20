import { useState } from 'react';
import { Eye, EyeOff, Loader2, CheckCircle } from 'lucide-react';
import { resetPassword } from '../utils/authService';

interface ResetPasswordScreenProps {
  token: string;
  onSuccess: () => void;
}

const ResetPasswordScreen = ({ token, onSuccess }: ResetPasswordScreenProps) => {
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);

    if (password !== confirmPassword) {
      setError('Пароли не совпадают');
      return;
    }
    if (password.length < 8) {
      setError('Пароль должен содержать не менее 8 символов');
      return;
    }

    setLoading(true);
    try {
      await resetPassword(token, password);
      setSuccess(true);
      setTimeout(() => {
        onSuccess();
      }, 1500);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : 'Произошла ошибка при сбросе пароля',
      );
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen bg-[#F7F8FA] flex items-center justify-center p-4">
      <div className="w-full max-w-md">
        <div className="flex flex-col items-center mb-8">
          <div className="w-16 h-16 rounded-2xl overflow-hidden flex items-center justify-center mb-4">
            <img src="/icon-192x192.png" alt="Personage" className="w-full h-full object-contain" />
          </div>
          <h1 className="text-2xl font-bold text-[#1A1B1E]">Personage</h1>
          <p className="text-sm text-gray-500 mt-1">Персональный ассистент</p>
        </div>

        <div className="bg-white rounded-2xl shadow-sm border border-gray-100 p-8">
          <div className="mb-6">
            <h2 className="text-lg font-semibold text-[#1A1B1E]">
              Новый пароль
            </h2>
            <p className="text-sm text-gray-500 mt-1">
              Придумайте надёжный пароль для вашего аккаунта
            </p>
          </div>

          {success ? (
            <div className="flex flex-col items-center py-4 gap-3">
              <div className="w-12 h-12 bg-green-100 rounded-full flex items-center justify-center">
                <CheckCircle size={24} className="text-green-600" />
              </div>
              <p className="text-sm font-medium text-green-700">
                Пароль успешно изменён!
              </p>
              <p className="text-xs text-gray-500">Выполняется вход...</p>
            </div>
          ) : (
            <form onSubmit={(e) => void handleSubmit(e)} className="space-y-4">
              {error && (
                <div className="p-3 rounded-lg bg-red-50 border border-red-100 text-sm text-red-600">
                  {error}
                </div>
              )}
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1.5">
                  Новый пароль
                </label>
                <div className="relative">
                  <input
                    type={showPassword ? 'text' : 'password'}
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    required
                    placeholder="••••••••"
                    autoComplete="new-password"
                    className="w-full px-4 py-2.5 pr-10 rounded-xl border border-gray-200 text-base focus:outline-none focus:ring-2 focus:ring-[#5C6BFF]/30 focus:border-[#5C6BFF] transition-all"
                  />
                  <button
                    type="button"
                    onClick={() => setShowPassword(!showPassword)}
                    className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600"
                  >
                    {showPassword ? <EyeOff size={16} /> : <Eye size={16} />}
                  </button>
                </div>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1.5">
                  Подтверждение пароля
                </label>
                <input
                  type={showPassword ? 'text' : 'password'}
                  value={confirmPassword}
                  onChange={(e) => setConfirmPassword(e.target.value)}
                  required
                  placeholder="••••••••"
                  autoComplete="new-password"
                  className="w-full px-4 py-2.5 rounded-xl border border-gray-200 text-base focus:outline-none focus:ring-2 focus:ring-[#5C6BFF]/30 focus:border-[#5C6BFF] transition-all"
                />
              </div>
              <button
                type="submit"
                disabled={loading}
                className="w-full py-2.5 rounded-xl bg-[#5C6BFF] text-white text-sm font-medium hover:bg-[#4B5AEE] disabled:opacity-60 disabled:cursor-not-allowed transition-all flex items-center justify-center gap-2"
              >
                {loading ? <Loader2 size={16} className="animate-spin" /> : null}
                {loading ? 'Сохранение...' : 'Сохранить пароль'}
              </button>
            </form>
          )}
        </div>
      </div>
    </div>
  );
};

export default ResetPasswordScreen;
