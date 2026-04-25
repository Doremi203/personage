export const AUTH_URL =
  process.env.VITE_AUTH_API_URL ?? 'https://auth.persomanage.ru';

export function requireCreds(): { email: string; password: string } {
  const email = process.env.TEST_USER_EMAIL;
  const password = process.env.TEST_USER_PASSWORD;
  if (!email || !password) {
    throw new Error(
      'TEST_USER_EMAIL and TEST_USER_PASSWORD must be set to run e2e tests against the live backend.',
    );
  }
  return { email, password };
}
