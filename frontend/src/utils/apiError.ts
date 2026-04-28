// Centralized API error handling. Parses error responses from all backends
// (Auth REST, gRPC-Gateway services, FastAPI telegram-auth) and produces
// user-friendly Russian messages keyed off the response status and, when
// available, the Auth API's `errorCode` field.

const CYRILLIC_RE = /[А-Яа-яЁё]/;

// Maps the Auth API's ErrorCode enum to user-friendly Russian copy.
// Source: backend/Personage.Auth/Personage.Auth.Domain/Exceptions/Base/ErrorCode.cs.
const AUTH_ERROR_MESSAGES: Record<string, string> = {
  InvalidCredentials: 'Неверный email или пароль.',
  UserAlreadyExists: 'Аккаунт с таким email уже существует.',
  UserNotFound: 'Пользователь не найден.',
  EmailValidationFail: 'Введите корректный email.',
  PasswordValidationFail: 'Пароль не отвечает требованиям безопасности.',
  UserNameValidationFail: 'Введите корректное имя.',
  PasswordNotSet: 'Для этого аккаунта вход по паролю недоступен.',
  InvalidResetToken: 'Ссылка для сброса пароля недействительна или истекла.',
  InvalidRefreshToken: 'Сессия истекла. Войдите заново.',
  TokenNotFound: 'Токен авторизации не найден.',
  TelegramSessionNotFound: 'Сессия Telegram не найдена.',
  ServiceTypeNotSupported: 'Этот тип интеграции не поддерживается.',
  OAuthError: 'Не удалось завершить авторизацию.',
  InvalidClaims: 'Сессия истекла. Войдите заново.',
  DuplicatedUsersForbidden: 'Не удалось обработать запрос: конфликт данных пользователя.',
  UsersNotAuthorizedForProcessing: 'Аккаунт ещё не подтверждён.',
};

export interface ApiErrorOptions {
  errorCode?: string | null;
  serverMessage?: string | null;
}

export class ApiError extends Error {
  readonly status: number;
  readonly errorCode: string | null;
  readonly serverMessage: string | null;

  constructor(status: number, message: string, options: ApiErrorOptions = {}) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
    this.errorCode = options.errorCode ?? null;
    this.serverMessage = options.serverMessage ?? null;
  }
}

export function statusFallback(status: number): string {
  if (status === 400) return 'Некорректный запрос. Проверьте введённые данные.';
  if (status === 401) return 'Сессия истекла. Войдите заново.';
  if (status === 403) return 'Недостаточно прав для этого действия.';
  if (status === 404) return 'Запрашиваемый объект не найден.';
  if (status === 405) return 'Действие не поддерживается.';
  if (status === 408) return 'Сервер не успел обработать запрос. Попробуйте ещё раз.';
  if (status === 409) return 'Конфликт данных. Возможно, объект уже изменён.';
  if (status === 410) return 'Запрашиваемый объект больше недоступен.';
  if (status === 413) return 'Слишком большой объём данных.';
  if (status === 415) return 'Неподдерживаемый формат данных.';
  if (status === 422) return 'Введённые данные не прошли проверку.';
  if (status === 429) return 'Слишком много запросов. Попробуйте позже.';
  if (status >= 400 && status < 500) return `Некорректный запрос (${status}).`;
  if (status >= 500 && status < 600) return 'Сервис временно недоступен. Попробуйте позже.';
  return `Ошибка: ${status}`;
}

interface ErrorBody {
  message?: unknown;
  Message?: unknown;
  detail?: unknown;
  error?: unknown;
  errorCode?: unknown;
}

function pickString(value: unknown): string | null {
  return typeof value === 'string' && value.trim() !== '' ? value : null;
}

async function parseErrorBody(response: Response): Promise<ErrorBody> {
  const ct = response.headers.get('content-type') ?? '';
  try {
    const text = await response.text();
    if (!text) return {};
    if (ct.includes('json') || text.trimStart().startsWith('{')) {
      try {
        return JSON.parse(text) as ErrorBody;
      } catch {
        return { message: text };
      }
    }
    return { message: text };
  } catch {
    return {};
  }
}

export async function readApiError(response: Response): Promise<ApiError> {
  const body = await parseErrorBody(response);
  const errorCode = pickString(body.errorCode);
  const serverMessage =
    pickString(body.message) ??
    pickString(body.Message) ??
    pickString(body.detail) ??
    pickString(body.error);

  if (errorCode && AUTH_ERROR_MESSAGES[errorCode]) {
    return new ApiError(response.status, AUTH_ERROR_MESSAGES[errorCode], {
      errorCode,
      serverMessage,
    });
  }

  if (serverMessage && CYRILLIC_RE.test(serverMessage)) {
    return new ApiError(response.status, serverMessage, { errorCode, serverMessage });
  }

  return new ApiError(response.status, statusFallback(response.status), {
    errorCode,
    serverMessage,
  });
}

export async function throwIfError(response: Response): Promise<void> {
  if (!response.ok) {
    throw await readApiError(response);
  }
}
