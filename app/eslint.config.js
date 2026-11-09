import js from '@eslint/js';
import reactHooks from 'eslint-plugin-react-hooks';
import reactRefresh from 'eslint-plugin-react-refresh';
import globals from 'globals';
import { defineConfig, globalIgnores } from 'eslint/config';
import tseslint from 'typescript-eslint';

export default defineConfig([
  globalIgnores(['dist', 'internal']),
  {
    files: ['**/*.{ts,tsx}'],
    extends: [js.configs.recommended, tseslint.configs.recommended, reactHooks.configs.flat['recommended-latest'], reactRefresh.configs.vite],
    languageOptions: {
      ecmaVersion: 2022,
      globals: globals.browser,
    },
    rules: {
      // This codebase colocates small pure helpers with the component that uses them
      // (e.g. BarList.tsx exports both `BarList` and `barItems`) for testability; that's
      // an intentional pattern here, not something Fast Refresh friendliness should drive.
      'react-refresh/only-export-components': 'off',
    },
  },
]);
