import { fileURLToPath } from 'node:url'
import { mergeConfig, defineConfig, configDefaults } from 'vitest/config'
import type { UserConfig as ViteUserConfig } from 'vite'
import viteConfig from './vite.config'

async function resolveViteConfig(): Promise<ViteUserConfig> {
  const resolved =
    typeof viteConfig === 'function'
      ? await viteConfig({
          command: 'serve',
          mode: 'test',
          isSsrBuild: false,
          isPreview: false,
        })
      : await viteConfig

  return resolved as ViteUserConfig
}

export default defineConfig(async () =>
  mergeConfig(await resolveViteConfig(), {
    test: {
      environment: 'jsdom',
      exclude: [...configDefaults.exclude, 'e2e/**'],
      root: fileURLToPath(new URL('./', import.meta.url)),
    },
  })
)
