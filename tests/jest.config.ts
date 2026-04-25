import type { Config } from 'jest';

const config: Config = {
  testTimeout: 15000,
  projects: [
    {
      displayName: 'tasker',
      preset: 'ts-jest',
      testEnvironment: 'node',
      roots: ['<rootDir>/tasker'],
      globalSetup: '<rootDir>/setup/global-setup.ts',
    },
    {
      displayName: 'notificator',
      preset: 'ts-jest',
      testEnvironment: 'node',
      roots: ['<rootDir>/notificator'],
      globalSetup: '<rootDir>/setup/notificator-setup.ts',
    },
  ],
};

export default config;
