<script setup lang="ts">
defineProps<{
  modelValue?: boolean
  required?: boolean
  locale?: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
}>()
</script>

<template>
  <div class="legal-consent" :class="{ 'is-required': required }" data-testid="legal-consent">
    <label v-if="required" class="consent-label">
      <input
        type="checkbox"
        :checked="modelValue"
        data-testid="legal-consent-checkbox"
        @change="emit('update:modelValue', ($event.target as HTMLInputElement).checked)"
      />
      <span>{{ locale?.toLowerCase().startsWith('zh') ? '我已阅读并同意' : 'I have read and agree to' }}</span>
    </label>
    <span v-else>{{ locale?.toLowerCase().startsWith('zh') ? '继续登录即表示你已阅读并同意' : 'By continuing, you acknowledge the' }}</span>
    <RouterLink to="/legal/terms" target="_blank">{{ locale?.toLowerCase().startsWith('zh') ? '《服务协议》' : 'Terms of Service' }}</RouterLink>
    <span>{{ locale?.toLowerCase().startsWith('zh') ? '和' : ' and ' }}</span>
    <RouterLink to="/legal/privacy" target="_blank">{{ locale?.toLowerCase().startsWith('zh') ? '《隐私政策》' : 'Privacy Policy' }}</RouterLink>
  </div>
</template>

<style scoped>
.legal-consent {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: center;
  gap: 3px;
  color: #7b8799;
  font-size: 12px;
  line-height: 1.7;
}

.legal-consent.is-required {
  justify-content: flex-start;
}

.consent-label {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  cursor: pointer;
}

.consent-label input {
  width: 15px;
  height: 15px;
  margin: 0;
  accent-color: #2563eb;
}

a {
  color: #1677ff;
  font-weight: 600;
  text-decoration: none;
}

a:hover {
  text-decoration: underline;
}
</style>
