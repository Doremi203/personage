const MAX_ATTEMPTS = 30;
const DELAY_MS = 1000;

export default async function globalSetup(): Promise<void> {
  const baseUrl = process.env.TASKER_URL ?? 'http://localhost:9090';

  for (let i = 0; i < MAX_ATTEMPTS; i++) {
    try {
      const res = await fetch(`${baseUrl}/health`);
      if (res.ok) {
        console.log(`Tasker service ready at ${baseUrl}`);
        return;
      }
    } catch {
      // service not yet up
    }
    await new Promise((resolve) => setTimeout(resolve, DELAY_MS));
  }

  throw new Error(
    `Tasker service at ${baseUrl} did not become healthy within ${MAX_ATTEMPTS}s`,
  );
}
