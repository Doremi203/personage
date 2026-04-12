import type { Config } from 'jest';

const config: Config = {
  preset: 'ts-jest',
  testEnvironment: 'node',
  rootDir: 'src',
  globalSetup: './setup/global-setup.ts',
  testTimeout: 15000,
};

export default config;
