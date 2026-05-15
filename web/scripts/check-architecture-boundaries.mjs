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
    pattern: /@\/architecture\/runtime\/types\/widget(?=\/|['"])|src\/architecture\/runtime\/types\/widget(?=\/|['"])/,
    message: 'widget component types must be imported from architecture/presentation/shared/types/widget',
  },
  {
    pattern: /@\/architecture\/runtime\/types\/widget-configs|src\/architecture\/runtime\/types\/widget-configs/,
    message: 'widget config types must be imported from architecture/domain/types/widget-configs',
  },
  {
    pattern: /@\/architecture\/runtime\/types\/chart|src\/architecture\/runtime\/types\/chart/,
    message: 'chart types must be imported from architecture/domain/types/chart',
  },
  {
    pattern: /@\/architecture\/runtime\/constants\/(?:widget|search|field|select)|src\/architecture\/runtime\/constants\/(?:widget|search|field|select)/,
    message: 'widget/search/field constants must be imported from architecture/domain/constants',
  },
  {
    pattern: /@\/architecture\/runtime\/utils\/(?:field|fieldSort|validationUtils|widgetOptionColors)|src\/architecture\/runtime\/utils\/(?:field|fieldSort|validationUtils|widgetOptionColors)/,
    message: 'field helper utilities must be imported from architecture/domain/utils',
  },
  {
    pattern: /@\/architecture\/runtime\/utils\/(?:searchValueNormalizer|searchParams|stringUtils|widgetConfigFlag)|src\/architecture\/runtime\/utils\/(?:searchValueNormalizer|searchParams|stringUtils|widgetConfigFlag)/,
    message: 'search and widget config utilities must be imported from architecture/domain/utils',
  },
  {
    pattern: /@\/architecture\/runtime\/utils\/(?:linkNavigation|routeSource|urlParams|queryParamKeys|queryParams|route)|src\/architecture\/runtime\/utils\/(?:linkNavigation|routeSource|urlParams|queryParamKeys|queryParams|route)/,
    message: 'routing helpers must be imported from architecture/shared/routing',
  },
  {
    pattern: /@\/architecture\/runtime\/(?:utils\/objectDiff|tableRuntime\/search)|src\/architecture\/runtime\/(?:utils\/objectDiff|tableRuntime\/search)/,
    message: 'object diff helpers must be imported from architecture/domain/utils/objectDiff',
  },
  {
    pattern: /@\/architecture\/runtime\/utils\/(?:serviceTreeUtils|tree-utils)|src\/architecture\/runtime\/utils\/(?:serviceTreeUtils|tree-utils)/,
    message: 'service tree helpers must be imported from architecture/domain/utils',
  },
  {
    pattern: /@\/architecture\/runtime\/utils\/(?:resourcePath|storagePreviewUrl)|src\/architecture\/runtime\/utils\/(?:resourcePath|storagePreviewUrl)/,
    message: 'resource path helpers must be imported from architecture/shared',
  },
  {
    pattern: /@\/architecture\/runtime\/utils\/directoryBundleFile|src\/architecture\/runtime\/utils\/directoryBundleFile/,
    message: 'directory bundle file helpers must be imported from architecture/presentation/utils/directoryBundleFile',
  },
  {
    pattern: /@\/architecture\/runtime\/utils\/(?:userInfo|tableUserInfo|permissionActors)|src\/architecture\/runtime\/utils\/(?:userInfo|tableUserInfo|permissionActors)/,
    message: 'user and permission helpers must be imported from architecture/domain/utils',
  },
  {
    pattern: /@\/architecture\/runtime\/utils\/functionTypes|src\/architecture\/runtime\/utils\/functionTypes/,
    message: 'function template type constants must be imported from architecture/domain/constants/functionTypes',
  },
  {
    pattern: /@\/architecture\/runtime\/utils\/(?:goPackageName|widgetFieldHelpers)|src\/architecture\/runtime\/utils\/(?:goPackageName|widgetFieldHelpers)/,
    message: 'domain helper utilities must be imported from architecture/domain/utils',
  },
  {
    pattern: /@\/architecture\/runtime\/utils\/(?:sanitizeHtml|clone)|src\/architecture\/runtime\/utils\/(?:sanitizeHtml|clone)/,
    message: 'shared browser-safe utilities must be imported from architecture/shared',
  },
  {
    pattern: /@\/architecture\/runtime\/utils\/(?:downloadZip|ErrorHandler)|src\/architecture\/runtime\/utils\/(?:downloadZip|ErrorHandler)/,
    message: 'presentation utilities must be imported from architecture/presentation/utils',
  },
  {
    pattern: /@\/architecture\/runtime\/utils\/zIndex|src\/architecture\/runtime\/utils\/zIndex/,
    message: 'presentation z-index constants must be imported from architecture/presentation/constants/zIndex',
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
    pattern: /@\/architecture\/(?:runtime|infrastructure)\/config\/features|src\/architecture\/(?:runtime|infrastructure)\/config\/features/,
    message: 'feature flags must be imported from architecture/shared/config/features',
  },
  {
    pattern: /@\/architecture\/runtime\/utils\/ExpressionParser(?:Adapter|V2)?|src\/architecture\/runtime\/utils\/ExpressionParser(?:Adapter|V2)?/,
    message: 'expression evaluators must be imported from architecture/domain/expression',
  },
  {
    pattern: /@\/architecture\/runtime\/stores\/extractors|src\/architecture\/runtime\/stores\/extractors/,
    message: 'field extractors must be imported from architecture/domain/form/extractors',
  },
  {
    pattern: /@\/architecture\/runtime\/stores\/(?:formData|responseData)|src\/architecture\/runtime\/stores\/(?:formData|responseData)/,
    message: 'Pinia form stores must be imported from architecture/infrastructure/stores',
  },
  {
    pattern: /@\/architecture\/runtime\/utils\/navigation|src\/architecture\/runtime\/utils\/navigation/,
    message: 'navigation port must be imported from architecture/shared/routing/navigation',
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

const removedRuntimePath = join(srcRoot, 'architecture', 'runtime')
if (existsSync(removedRuntimePath)) {
  failures.push(`${relative(root, removedRuntimePath)} has been removed; use domain, shared, infrastructure, or presentation`)
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

    const normalizedFile = normalizePath(relative(root, file))
    if (
      normalizedFile.startsWith('src/architecture/presentation/')
      && normalizedFile !== 'src/architecture/presentation/context/formRuntimeContext.ts'
      && /@\/architecture\/infrastructure\/stores\/(?:formData|responseData)/.test(line)
    ) {
      failures.push(`${relative(root, file)}:${index + 1} presentation form runtime must use presentation/context/formRuntimeContext: ${line.trim()}`)
    }

    if (
      normalizedFile.startsWith('src/architecture/presentation/')
      && normalizedFile !== 'src/architecture/presentation/context/appStoresContext.ts'
      && /@\/architecture\/infrastructure\/stores\/(?:auth|departmentInfo|license|theme|userInfo)/.test(line)
    ) {
      failures.push(`${relative(root, file)}:${index + 1} presentation app stores must use presentation/context/appStoresContext: ${line.trim()}`)
    }

    if (
      normalizedFile.startsWith('src/architecture/presentation/')
      && normalizedFile !== 'src/architecture/presentation/context/eventBusContext.ts'
      && /@\/architecture\/infrastructure\/eventBus/.test(line)
    ) {
      failures.push(`${relative(root, file)}:${index + 1} presentation event bus must use presentation/context/eventBusContext: ${line.trim()}`)
    }

    if (
      normalizedFile.startsWith('src/architecture/presentation/')
      && normalizedFile !== 'src/architecture/presentation/context/uploadContext.ts'
      && /@\/architecture\/infrastructure\/upload/.test(line)
    ) {
      failures.push(`${relative(root, file)}:${index + 1} presentation upload APIs must use presentation/context/uploadContext: ${line.trim()}`)
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
