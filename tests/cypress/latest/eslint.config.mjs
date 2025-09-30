import typescriptEslint from '@typescript-eslint/eslint-plugin';
import tsParser from '@typescript-eslint/parser';
import pluginCypress from 'eslint-plugin-cypress';
import pluginPrettier from 'eslint-plugin-prettier';
import prettierConfig from 'eslint-config-prettier';

export default [
  pluginCypress.configs.recommended,

  {
    plugins: {
      '@typescript-eslint': typescriptEslint,
      cypress: pluginCypress,
      prettier: pluginPrettier,
    },

    languageOptions: {
      parser: tsParser,
      ecmaVersion: 2021,
      sourceType: 'module',
    },

    rules: {
      'prettier/prettier': [
        'warn',
        {
          semi: true,
          singleQuote: true,
          tabWidth: 2,
          useTabs: false,
        },
      ],
      indent: ['warn', 2],
      'eol-last': ['error', 'always'],
      'cypress/no-unnecessary-waiting': 'off',
      '@typescript-eslint/no-explicit-any': 'off',
      'cypress/unsafe-to-chain-command': 'off',
    },
  },
];
