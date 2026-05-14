import { existsSync } from 'node:fs'
import { readdir, readFile } from 'node:fs/promises'
import { join, relative } from 'node:path'

const root = new URL('..', import.meta.url).pathname
const srcRoot = join(root, 'src')

const forbiddenTopLevelDirs = [
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

const scannedFiles = [
  join(root, 'components.d.ts'),
  join(root, 'eslint.config.ts'),
  join(root, 'package.json'),
  join(root, 'README.md'),
  join(root, 'tsconfig.app.json'),
  join(root, 'tsconfig.json'),
  join(root, 'tsconfig.node.json'),
  join(root, 'tsconfig.vitest.json'),
  join(root, 'vite.config.ts'),
  join(root, 'vitest.config.ts'),
]

const scannedExtensions = new Set(['.ts', '.tsx', '.vue', '.md', '.scss', '.css', '.d.ts'])
const codeExtensions = new Set(['.ts', '.tsx', '.vue', '.d.ts'])

const forbiddenPatterns = [
  {
    pattern: /@\/(?:api|assets|components|composables|config|core|features|router|shared|stores|styles|types|utils|views)(?=\/|['"])/,
    message: 'old @ alias import',
  },
  {
    pattern: /src\/(?:api|assets|components|composables|config|core|features|router|shared|stores|styles|types|utils|views)\b/,
    message: 'old src directory reference',
  },
  {
    pattern: /stores-v2|formData-v2|responseData-v2|Widgets-v2/,
    message: 'removed v2 migration reference',
  },
  {
    pattern: /@\/architecture\/runtime\/types\/field|src\/architecture\/runtime\/types\/field/,
    message: 'field types must be imported from architecture/domain/types/field',
  },
  {
    pattern: /@\/architecture\/runtime\/utils\/logger|src\/architecture\/runtime\/utils\/logger/,
    message: 'logger must be imported from architecture/shared/logger',
  },
  {
    pattern: /@\/architecture\/runtime\/utils\/functionSchemaSelectors|src\/architecture\/runtime\/utils\/functionSchemaSelectors/,
    message: 'schema selectors must be imported from architecture/domain/utils/functionSchemaSelectors',
  },
  {
    pattern: /@\/architecture\/runtime\/utils\/apiError|src\/architecture\/runtime\/utils\/apiError/,
    message: 'API error helpers must be imported from architecture/shared/apiError',
  },
  {
    pattern: /@\/architecture\/runtime\/utils\/searchFieldValue|src\/architecture\/runtime\/utils\/searchFieldValue/,
    message: 'search field value helpers must be imported from architecture/domain/utils/searchFieldValue',
  },
  {
    pattern: /@\/architecture\/runtime\/utils\/createFieldValue|src\/architecture\/runtime\/utils\/createFieldValue/,
    message: 'field value factories must be imported from architecture/domain/utils/createFieldValue',
  },
  {
    pattern: /@\/architecture\/runtime\/utils\/date|src\/architecture\/runtime\/utils\/date/,
    message: 'date helpers must be imported from architecture/shared/date',
  },
  {
    pattern: /@\/architecture\/runtime\/widgetRuntime\/defaultValue|src\/architecture\/runtime\/widgetRuntime\/defaultValue/,
    message: 'default value helpers must be imported from architecture/domain/utils/defaultValue',
  },
  {
    pattern: /@\/architecture\/runtime\/widgetRuntime\/dynamicDefaultValue|src\/architecture\/runtime\/widgetRuntime\/dynamicDefaultValue/,
    message: 'dynamic default value helpers must be imported from architecture/domain/utils/dynamicDefaultValue',
  },
  {
    pattern: /@\/architecture\/runtime\/widgetRuntime\/(?:fieldReset|dependency|presenceEffects)|src\/architecture\/runtime\/widgetRuntime\/(?:fieldReset|dependency|presenceEffects)/,
    message: 'field effect helpers must be imported from architecture/domain/utils',
  },
  {
    pattern: /@\/architecture\/runtime\/widgetRuntime\/validation|src\/architecture\/runtime\/widgetRuntime\/validation/,
    message: 'widget validation helpers must be imported from architecture/domain/utils/widgetValidation',
  },
  {
    pattern: /@\/architecture\/runtime\/widgetRuntime\/(?:containerValue|persistedFieldValue)|src\/architecture\/runtime\/widgetRuntime\/(?:containerValue|persistedFieldValue)/,
    message: 'widget value helpers must be imported from architecture/domain/utils',
  },
  {
    pattern: /@\/architecture\/runtime\/utils\/conditionEvaluator|src\/architecture\/runtime\/utils\/conditionEvaluator/,
    message: 'condition evaluator must be imported from architecture/domain/utils/conditionEvaluator',
  },
  {
    pattern: /@\/architecture\/runtime\/validation|src\/architecture\/runtime\/validation/,
    message: 'validation must be imported from architecture/domain/validation',
  },
  {
    pattern: /src\/components|src\/shared\/components/,
    message: 'old component auto-scan directory',
  },
  {
    pattern: /@\/architecture\/infrastructure\/config\/features|src\/architecture\/infrastructure\/config\/features/,
    message: 'feature flags must live in architecture/runtime/config',
  },
  {
    pattern: /import\s+type\s+\{[^}]*\b(?:FormState|ValidationResult|TableState|TableRow|TableResponse|SearchParams|SortParams|SortItem|WorkspaceState)\b[^}]*\}\s+from\s+['"][^'"]*domain\/services\//,
    message: 'domain state/data types must be imported from architecture/domain/types',
  },
]

const layerRules = [
  {
    root: 'src/architecture/domain/',
    forbidden: ['infrastructure', 'presentation'],
    message: 'domain layer must not depend on infrastructure or presentation',
    patterns: [
      {
        pattern: /\/(?:workspace|agent|hr|storage|control|message)\/api\/v\d+\//,
        message: 'domain layer must not hard-code backend API routes',
      },
    ],
  },
  {
    root: 'src/architecture/runtime/',
    forbidden: ['infrastructure', 'presentation'],
    message: 'runtime layer must not depend on infrastructure or presentation',
  },
  {
    root: 'src/architecture/application/',
    forbidden: ['presentation'],
    message: 'application layer must not depend on presentation',
  },
  {
    root: 'src/architecture/infrastructure/',
    forbidden: ['presentation'],
    message: 'infrastructure layer must not depend on presentation',
  },
  {
    root: 'src/architecture/presentation/',
    forbidden: [],
    message: 'presentation layer must not depend on transport details',
    patterns: [
      {
        pattern: /@\/architecture\/infrastructure\/apiClient|(?:\.\.\/)+infrastructure\/apiClient|\bgetApiClient\s*\(/,
        message: 'presentation layer must use gateways or API modules instead of raw apiClient',
      },
      {
        pattern: /['"]\/(?:workspace|agent|hr|storage|control|message)\/api\/v\d+\//,
        message: 'presentation layer must not hard-code backend API routes',
      },
    ],
  },
]

const failures = []

for (const dir of forbiddenTopLevelDirs) {
  const path = join(srcRoot, dir)
  if (existsSync(path)) {
    failures.push(`${relative(root, path)} should live under src/architecture`)
  }
}

function extensionOf(file) {
  if (file.endsWith('.d.ts')) return '.d.ts'
  const match = file.match(/\.[^.]+$/)
  return match?.[0] ?? ''
}

function normalizePath(file) {
  return file.split('\\').join('/')
}

function isTestFile(file) {
  return /\.test\.[tj]sx?$/.test(file) || file.includes('/__tests__/')
}

function layerRuleForFile(file) {
  const normalized = normalizePath(relative(root, file))
  return layerRules.find((rule) => normalized.startsWith(rule.root))
}

async function collectFiles(dir) {
  const entries = await readdir(dir, { withFileTypes: true })
  const files = []

  for (const entry of entries) {
    if (entry.name === 'node_modules' || entry.name === 'dist' || entry.name === '.vite') {
      continue
    }

    const path = join(dir, entry.name)
    if (entry.isDirectory()) {
      files.push(...await collectFiles(path))
    } else if (scannedExtensions.has(extensionOf(entry.name))) {
      files.push(path)
    }
  }

  return files
}

const files = [
  ...scannedFiles.filter((file) => existsSync(file)),
  ...await collectFiles(srcRoot),
]

for (const file of files) {
  const content = await readFile(file, 'utf8')
  const lines = content.split(/\r?\n/)

  lines.forEach((line, index) => {
    for (const { pattern, message } of forbiddenPatterns) {
      if (pattern.test(line)) {
        failures.push(`${relative(root, file)}:${index + 1} ${message}: ${line.trim()}`)
      }
    }

    const layerRule = layerRuleForFile(file)
    if (
      layerRule
      && codeExtensions.has(extensionOf(file))
      && !isTestFile(normalizePath(file))
    ) {
      for (const layer of layerRule.forbidden) {
        const pattern = new RegExp(`@/architecture/${layer}(?=/|['"])`)
        if (pattern.test(line)) {
          failures.push(`${relative(root, file)}:${index + 1} ${layerRule.message}: ${line.trim()}`)
        }
      }

      for (const { pattern, message } of layerRule.patterns ?? []) {
        if (pattern.test(line)) {
          failures.push(`${relative(root, file)}:${index + 1} ${message}: ${line.trim()}`)
        }
      }
    }
  })
}

if (failures.length > 0) {
  console.error('Architecture boundary check failed:')
  failures.forEach((failure) => console.error(`- ${failure}`))
  process.exit(1)
}

console.log('Architecture boundary check passed.')
