import { describe, expect, it } from 'vitest'
import { getLLMEndpointHost, resolveLLMProviderBrand } from './llmProviderBrands'

describe('resolveLLMProviderBrand', () => {
  it.each([
    ['gpt-5.5', '', 'openai'],
    ['qwen3.7-plus', '', 'qwen'],
    ['deepseek-chat', '', 'deepseek'],
    ['MiniMax-M2.7-highspeed', '', 'minimax'],
    ['claude-sonnet-4-5', '', 'anthropic'],
    ['gemini-2.5-pro', '', 'gemini'],
    ['kimi-k2', '', 'kimi'],
    ['mimo-v2.5-pro', '', 'xiaomi_mimo'],
    ['glm-4.5', '', 'zhipu'],
    ['custom-model', 'https://openrouter.ai/api/v1', 'openrouter']
  ])('identifies %s as %s', (model, apiBase, expected) => {
    expect(resolveLLMProviderBrand({ model, api_base: apiBase }).id).toBe(expected)
  })

  it('does not mistake an unknown OpenAI-compatible endpoint for OpenAI', () => {
    expect(resolveLLMProviderBrand({ provider: 'openai', model: 'company-model' }).id).toBe('compatible')
  })
})

describe('getLLMEndpointHost', () => {
  it('returns a compact source host', () => {
    expect(getLLMEndpointHost('https://api.example.com/v1')).toBe('api.example.com')
  })
})
