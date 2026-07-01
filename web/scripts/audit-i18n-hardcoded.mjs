import { readdir, readFile } from 'node:fs/promises'
import { join, relative, extname } from 'node:path'

const root = new URL('..', import.meta.url).pathname
const srcRoot = join(root, 'src')
const sourceExtensions = new Set(['.vue', '.ts', '.tsx', '.js'])
const cjkPattern = /[\u3400-\u9fff]/

const ignoredPathParts = [
  '/architecture/shared/i18n/locales/',
  '.test.',
  '.spec.',
  '/node_modules/',
]

const args = new Set(process.argv.slice(2))
const failOnFindings = args.has('--fail-on-findings')
const maxFiles = Number([...args].find(arg => arg.startsWith('--max-files='))?.split('=')[1] || 80)
const maxLinesPerFile = Number([...args].find(arg => arg.startsWith('--max-lines='))?.split('=')[1] || 5)

async function collectFiles(dir, files = []) {
  const entries = await readdir(dir, { withFileTypes: true })
  for (const entry of entries) {
    if (entry.name === 'node_modules') continue
    const fullPath = join(dir, entry.name)
    if (entry.isDirectory()) {
      await collectFiles(fullPath, files)
      continue
    }
    if (sourceExtensions.has(extname(entry.name))) {
      files.push(fullPath)
    }
  }
  return files
}

function stripComments(text) {
  return text
    .replace(/<!--[\s\S]*?-->/g, '')
    .replace(/\/\*[\s\S]*?\*\//g, '')
}

function stripLineComment(line) {
  let quote = ''
  let escaped = false

  for (let index = 0; index < line.length - 1; index += 1) {
    const char = line[index]
    const next = line[index + 1]

    if (escaped) {
      escaped = false
      continue
    }

    if (char === '\\') {
      escaped = true
      continue
    }

    if (quote) {
      if (char === quote) {
        quote = ''
      }
      continue
    }

    if (char === '"' || char === "'" || char === '`') {
      quote = char
      continue
    }

    if (char === '/' && next === '/') {
      return line.slice(0, index)
    }
  }

  return line
}

function shouldIgnoreFile(file) {
  const normalized = `/${relative(root, file).replaceAll('\\', '/')}`
  return ignoredPathParts.some(part => normalized.includes(part))
}

function shouldIgnoreLine(line) {
  const trimmed = line.trim()
  if (!trimmed) return true
  if (trimmed.startsWith('//') || trimmed.startsWith('*')) return true
  if (/src=(["'])[^"']*[\u3400-\u9fff][^"']*\.(?:svg|png|jpe?g|webp|gif)\1/.test(trimmed)) return true
  if (/^\s*import\s/.test(trimmed)) return true
  if (/^\s*type\s/.test(trimmed)) return true
  if (/^\s*interface\s/.test(trimmed)) return true
  return false
}

function findHardcodedLines(text) {
  const stripped = stripComments(text)
  return stripped
    .split(/\r?\n/)
    .map((text, index) => ({ line: index + 1, text: stripLineComment(text).trim() }))
    .filter(({ text }) => cjkPattern.test(text))
    .filter(({ text }) => !shouldIgnoreLine(text))
}

const files = (await collectFiles(srcRoot)).filter(file => !shouldIgnoreFile(file))
const findings = []

for (const file of files) {
  const text = await readFile(file, 'utf8')
  const hits = findHardcodedLines(text)
  if (hits.length > 0) {
    findings.push({ file, hits })
  }
}

findings.sort((left, right) => right.hits.length - left.hits.length || left.file.localeCompare(right.file))

const totalLines = findings.reduce((sum, item) => sum + item.hits.length, 0)
console.log(`i18n hardcoded CJK audit: ${findings.length} files, ${totalLines} lines`)

for (const finding of findings.slice(0, maxFiles)) {
  console.log(`\n${finding.hits.length}\t${relative(root, finding.file)}`)
  for (const hit of finding.hits.slice(0, maxLinesPerFile)) {
    console.log(`  ${hit.line}: ${hit.text.slice(0, 180)}`)
  }
  if (finding.hits.length > maxLinesPerFile) {
    console.log(`  ... +${finding.hits.length - maxLinesPerFile}`)
  }
}

if (findings.length > maxFiles) {
  console.log(`\n... ${findings.length - maxFiles} more files omitted. Increase --max-files to inspect more.`)
}

if (failOnFindings && findings.length > 0) {
  process.exitCode = 1
}
