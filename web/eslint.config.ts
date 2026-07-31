import { globalIgnores } from 'eslint/config'
import { defineConfigWithVueTs, vueTsConfigs } from '@vue/eslint-config-typescript'
import pluginVue from 'eslint-plugin-vue'
import pluginVitest from '@vitest/eslint-plugin'

// To allow more languages other than `ts` in `.vue` files, uncomment the following lines:
// import { configureVueProject } from '@vue/eslint-config-typescript'
// configureVueProject({ scriptLangs: ['ts', 'tsx'] })
// More info at https://github.com/vuejs/eslint-config-typescript/#advanced-setup

const removedTopLevelAliases = [
  'api',
  'assets',
  'components',
  'composables',
  'config',
  'core',
  'features',
  'router',
  'shared',
  'stores',
  'styles',
  'types',
  'utils',
  'views',
]

export default defineConfigWithVueTs(
  {
    name: 'app/files-to-lint',
    files: ['**/*.{ts,mts,tsx,vue}'],
  },

  globalIgnores([
    '**/dist/**',
    '**/dist-ssr/**',
    '**/coverage/**',
    '**/playwright-report/**',
    '**/test-results/**',
    '**/.playwright-mcp/**',
  ]),

  pluginVue.configs['flat/essential'],
  vueTsConfigs.recommended,
  
  {
    ...pluginVitest.configs.recommended,
    files: ['src/**/__tests__/*'],
  },

  {
    name: 'app/architecture-boundaries',
    rules: {
      // Open-source baseline: keep architectural boundary violations blocking,
      // while older Vue/TypeScript migration debt is warning-only until it is
      // paid down file by file.
      '@typescript-eslint/no-explicit-any': 'off',
      '@typescript-eslint/no-empty-object-type': 'off',
      '@typescript-eslint/no-namespace': 'off',
      '@typescript-eslint/no-unsafe-function-type': 'warn',
      '@typescript-eslint/no-unused-expressions': 'warn',
      '@typescript-eslint/no-unused-vars': ['warn', {
        argsIgnorePattern: '^_',
        varsIgnorePattern: '^_',
        caughtErrorsIgnorePattern: '^_',
        ignoreRestSiblings: true,
      }],
      'prefer-const': 'warn',
      'vue/multi-word-component-names': 'off',
      'vue/no-deprecated-filter': 'warn',
      'vue/no-deprecated-v-on-native-modifier': 'warn',
      'vue/no-dupe-keys': 'warn',
      'vue/no-mutating-props': 'warn',
      'vue/no-side-effects-in-computed-properties': 'warn',
      'vue/no-unused-vars': 'warn',
      'no-restricted-imports': [
        'error',
        {
          patterns: [
            {
              group: removedTopLevelAliases.map((alias) => `@/${alias}/*`),
              message: 'Use the unified src/architecture/* path instead of old top-level frontend aliases.',
            },
          ],
        },
      ],
    },
  },
)
