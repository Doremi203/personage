/**
 * Builds a JWT-shaped token accepted by the dev/test stub verifier.
 * The stub verifier only checks that the token has 3 dot-separated parts
 * and decodes the payload to extract `user_id` — no cryptographic check.
 */
export function makeToken(userId: string): string {
  const header  = Buffer.from(JSON.stringify({ alg: 'HS256', typ: 'JWT' })).toString('base64url');
  const payload = Buffer.from(JSON.stringify({ user_id: userId })).toString('base64url');
  return `${header}.${payload}.test-signature`;
}
