import anthropicLogo from '@lobehub/icons-static-svg/icons/anthropic.svg'
import azureLogo from '@lobehub/icons-static-svg/icons/azure-color.svg'
import baiduLogo from '@lobehub/icons-static-svg/icons/baidu-color.svg'
import bedrockLogo from '@lobehub/icons-static-svg/icons/bedrock-color.svg'
import cohereLogo from '@lobehub/icons-static-svg/icons/cohere-color.svg'
import deepseekLogo from '@lobehub/icons-static-svg/icons/deepseek-color.svg'
import doubaoLogo from '@lobehub/icons-static-svg/icons/doubao-color.svg'
import geminiLogo from '@lobehub/icons-static-svg/icons/gemini-color.svg'
import grokLogo from '@lobehub/icons-static-svg/icons/grok.svg'
import hunyuanLogo from '@lobehub/icons-static-svg/icons/hunyuan-color.svg'
import metaLogo from '@lobehub/icons-static-svg/icons/meta-color.svg'
import minimaxLogo from '@lobehub/icons-static-svg/icons/minimax-color.svg'
import mistralLogo from '@lobehub/icons-static-svg/icons/mistral-color.svg'
import moonshotLogo from '@lobehub/icons-static-svg/icons/moonshot.svg'
import ollamaLogo from '@lobehub/icons-static-svg/icons/ollama.svg'
import openaiLogo from '@lobehub/icons-static-svg/icons/openai.svg'
import openrouterLogo from '@lobehub/icons-static-svg/icons/openrouter-color.svg'
import perplexityLogo from '@lobehub/icons-static-svg/icons/perplexity-color.svg'
import qwenLogo from '@lobehub/icons-static-svg/icons/qwen-color.svg'
import siliconCloudLogo from '@lobehub/icons-static-svg/icons/siliconcloud-color.svg'
import togetherLogo from '@lobehub/icons-static-svg/icons/together-color.svg'
import yiLogo from '@lobehub/icons-static-svg/icons/yi-color.svg'
import xiaomiMiMoLogo from '@lobehub/icons-static-svg/icons/xiaomimimo.svg'
import zhipuLogo from '@lobehub/icons-static-svg/icons/zai.svg'
import kimiLogo from './provider-logos/kimi.svg'

export interface LLMProviderBrand {
  id: string
  name: string
  logo?: string
  accent: string
  surface: string
}

export interface LLMProviderBrandSource {
  name?: string
  provider?: string
  protocol?: string
  model?: string
  api_base?: string
}

const brands: Record<string, LLMProviderBrand> = {
  openai: { id: 'openai', name: 'OpenAI', logo: openaiLogo, accent: '#111827', surface: '#f3f4f6' },
  anthropic: { id: 'anthropic', name: 'Anthropic', logo: anthropicLogo, accent: '#c15f3c', surface: '#fdf1eb' },
  deepseek: { id: 'deepseek', name: 'DeepSeek', logo: deepseekLogo, accent: '#4d6bfe', surface: '#eef2ff' },
  qwen: { id: 'qwen', name: '通义千问', logo: qwenLogo, accent: '#615ced', surface: '#f0efff' },
  minimax: { id: 'minimax', name: 'MiniMax', logo: minimaxLogo, accent: '#ef594d', surface: '#fff0ee' },
  gemini: { id: 'gemini', name: 'Google Gemini', logo: geminiLogo, accent: '#4285f4', surface: '#edf5ff' },
  kimi: { id: 'kimi', name: 'Kimi', logo: kimiLogo, accent: '#1783ff', surface: '#edf6ff' },
  moonshot: { id: 'moonshot', name: 'Moonshot AI', logo: moonshotLogo, accent: '#111827', surface: '#f3f4f6' },
  zhipu: { id: 'zhipu', name: '智谱 AI', logo: zhipuLogo, accent: '#111827', surface: '#f3f4f6' },
  mistral: { id: 'mistral', name: 'Mistral AI', logo: mistralLogo, accent: '#f97316', surface: '#fff3e8' },
  grok: { id: 'grok', name: 'xAI', logo: grokLogo, accent: '#111827', surface: '#f3f4f6' },
  meta: { id: 'meta', name: 'Meta', logo: metaLogo, accent: '#0866ff', surface: '#edf5ff' },
  azure: { id: 'azure', name: 'Azure AI', logo: azureLogo, accent: '#0078d4', surface: '#eaf5ff' },
  bedrock: { id: 'bedrock', name: 'Amazon Bedrock', logo: bedrockLogo, accent: '#ff9900', surface: '#fff4df' },
  doubao: { id: 'doubao', name: '豆包', logo: doubaoLogo, accent: '#2f6bff', surface: '#edf3ff' },
  hunyuan: { id: 'hunyuan', name: '腾讯混元', logo: hunyuanLogo, accent: '#006eff', surface: '#edf5ff' },
  baidu: { id: 'baidu', name: '百度千帆', logo: baiduLogo, accent: '#2932e1', surface: '#eef0ff' },
  yi: { id: 'yi', name: '零一万物', logo: yiLogo, accent: '#7c3aed', surface: '#f3eeff' },
  xiaomi_mimo: { id: 'xiaomi_mimo', name: '小米 MiMo', logo: xiaomiMiMoLogo, accent: '#ff6900', surface: '#fff3e8' },
  ollama: { id: 'ollama', name: 'Ollama', logo: ollamaLogo, accent: '#111827', surface: '#f3f4f6' },
  siliconflow: { id: 'siliconflow', name: 'SiliconFlow', logo: siliconCloudLogo, accent: '#7c3aed', surface: '#f3eeff' },
  openrouter: { id: 'openrouter', name: 'OpenRouter', logo: openrouterLogo, accent: '#5b45ff', surface: '#f0efff' },
  cohere: { id: 'cohere', name: 'Cohere', logo: cohereLogo, accent: '#39594d', surface: '#edf5f1' },
  perplexity: { id: 'perplexity', name: 'Perplexity', logo: perplexityLogo, accent: '#20808d', surface: '#eaf6f7' },
  together: { id: 'together', name: 'Together AI', logo: togetherLogo, accent: '#111827', surface: '#f3f4f6' },
  compatible: { id: 'compatible', name: 'OpenAI 兼容', accent: '#64748b', surface: '#f1f5f9' }
}

const brandMatchers: Array<[keyof typeof brands, RegExp]> = [
  ['ollama', /\bollama\b|localhost:11434|127\.0\.0\.1:11434/],
  ['openrouter', /openrouter/],
  ['siliconflow', /siliconflow|siliconcloud/],
  ['together', /together\.ai|\btogether\b/],
  ['azure', /azure|openai\.azure\.com/],
  ['bedrock', /bedrock|amazonaws/],
  ['deepseek', /deepseek/],
  ['qwen', /qwen|dashscope|aliyuncs|alibaba/],
  ['minimax', /minimax|abab\d/],
  ['xiaomi_mimo', /xiaomi.?mimo|\bmimo[-_\d]/],
  ['gemini', /gemini|generativelanguage\.googleapis|vertexai/],
  ['anthropic', /anthropic|claude/],
  ['kimi', /\bkimi(?:[-_\d]|\b)/],
  ['moonshot', /moonshot/],
  ['zhipu', /zhipu|bigmodel|chatglm|\bglm[-_\d]/],
  ['mistral', /mistral|codestral/],
  ['grok', /\bgrok\b|api\.x\.ai/],
  ['doubao', /doubao|volcengine|volces|bytedance/],
  ['hunyuan', /hunyuan|tencentcloud/],
  ['baidu', /ernie|qianfan|baidu/],
  ['yi', /\byi[-_\d]|01\.ai|lingyiwanwu/],
  ['cohere', /cohere|command-r/],
  ['perplexity', /perplexity|sonar-(?:small|medium|large|pro)/],
  ['meta', /\bllama\b|meta-ai/],
  ['openai', /api\.openai\.com|\bopenai\b|\bchatgpt\b|\bgpt[-_\d]|\bo[134][-_\d]/]
]

export function resolveLLMProviderBrand(source: LLMProviderBrandSource): LLMProviderBrand {
  // provider is the transport family in kageos (for example, "openai" means
  // OpenAI-compatible), so only product-facing fields may identify the vendor.
  const haystack = [source.name, source.model, source.api_base]
    .filter(Boolean)
    .join(' ')
    .toLowerCase()

  for (const [brandId, matcher] of brandMatchers) {
    if (matcher.test(haystack)) return brands[brandId]
  }

  if (source.provider?.toLowerCase() === 'anthropic' || source.protocol?.toLowerCase() === 'anthropic_messages') {
    return brands.anthropic
  }

  return brands.compatible
}

export function getLLMEndpointHost(apiBase?: string): string {
  const value = apiBase?.trim()
  if (!value) return ''

  try {
    return new URL(value).host
  } catch {
    return value.replace(/^https?:\/\//, '').split('/')[0]
  }
}
