type EnvFlagValue = boolean | string | number | undefined

function readBooleanEnv(name: string, fallback: boolean): boolean {
  const value = (import.meta.env[name] as EnvFlagValue)
  if (value === undefined || value === '') {
    return fallback
  }
  if (typeof value === 'boolean') {
    return value
  }
  const normalized = String(value).trim().toLowerCase()
  if (['1', 'true', 'yes', 'on'].includes(normalized)) {
    return true
  }
  if (['0', 'false', 'no', 'off'].includes(normalized)) {
    return false
  }
  return fallback
}

const focusedMode = readBooleanEnv('VITE_AOS_FOCUSED_MODE', import.meta.env.MODE === 'test' ? false : true)

export const featureFlags = {
  focusedMode,
  operateLogs: readBooleanEnv('VITE_AOS_FEATURE_OPERATE_LOGS', true),
  capabilityBundle: readBooleanEnv('VITE_AOS_FEATURE_CAPABILITY_BUNDLE', true),
  docs: readBooleanEnv('VITE_AOS_FEATURE_DOCS', true),
  llmManagement: readBooleanEnv('VITE_AOS_FEATURE_LLM_MANAGEMENT', true),
  scheduledTasks: readBooleanEnv('VITE_AOS_FEATURE_SCHEDULED_TASKS', true)
} as const

export type FeatureKey = keyof typeof featureFlags

export function isFeatureEnabled(feature: FeatureKey): boolean {
  return featureFlags[feature]
}
