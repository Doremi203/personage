import type { Config } from 'jest';

const config: Config = {
  preset: 'ts-jest',
  testEnvironment: 'node',
  roots: ['<rootDir>/tasker'],
  globalSetup: './setup/global-setup.ts',
  testTimeout: 15000,
};

export default config;
