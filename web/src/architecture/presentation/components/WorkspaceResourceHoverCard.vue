<template>
  <Teleport to="body">
    <Transition name="workspace-resource-hover-card-fade">
      <section
        v-if="preview.visible"
        class="workspace-resource-hover-card"
        :class="`is-${preview.kind}`"
        :style="{ left: `${preview.left}px`, top: `${preview.top}px` }"
        role="dialog"
        :aria-label="ariaLabel"
        @mouseenter="emit('mouseenter')"
        @mouseleave="emit('mouseleave')"
        @focusin="emit('focusin')"
        @focusout="emit('focusout')"
      >
        <div class="workspace-resource-hover-card__head">
          <span class="workspace-resource-hover-card__icon">
            <img
              v-if="preview.iconSrc"
              :src="preview.iconSrc"
              :alt="preview.typeLabel"
              class="workspace-resource-hover-card__img"
            />
            <span
              v-else
              class="workspace-resource-hover-card__html-icon"
              aria-hidden="true"
              v-html="preview.iconHtml"
            ></span>
          </span>
          <span class="workspace-resource-hover-card__title-wrap">
            <span class="workspace-resource-hover-card__title">{{ preview.label }}</span>
            <span class="workspace-resource-hover-card__type">{{ preview.typeLabel }}</span>
          </span>
          <button
            type="button"
            class="workspace-resource-hover-card__close"
            aria-label="关闭资源预览"
            @click="emit('close')"
          >
            ×
          </button>
        </div>

        <p v-if="preview.description" class="workspace-resource-hover-card__description">
          {{ preview.description }}
        </p>
        <div v-if="preview.metaItems.length" class="workspace-resource-hover-card__meta">
          <span
            v-for="item in preview.metaItems"
            :key="item"
            class="workspace-resource-hover-card__meta-item"
          >
            {{ item }}
          </span>
        </div>
        <code v-if="preview.path" class="workspace-resource-hover-card__path">{{ preview.path }}</code>
        <div v-if="preview.loading" class="workspace-resource-hover-card__loading">正在加载详情...</div>
        <div class="workspace-resource-hover-card__actions">
          <button
            v-if="!isWorkspaceToolResourcePath(preview.path)"
            type="button"
            class="workspace-resource-hover-card__open"
            @click="emit('open')"
          >
            打开
          </button>
        </div>
      </section>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import type { WorkspaceResourcePreview } from '@/architecture/presentation/composables/useWorkspaceResourceHoverPreview'
import { isWorkspaceToolResourcePath } from '@/architecture/presentation/components/utils/workspaceInvocationSnippet'

withDefaults(defineProps<{
  preview: WorkspaceResourcePreview
  ariaLabel?: string
}>(), {
  ariaLabel: '资源预览',
})

const emit = defineEmits<{
  (e: 'mouseenter'): void
  (e: 'mouseleave'): void
  (e: 'focusin'): void
  (e: 'focusout'): void
  (e: 'open'): void
  (e: 'close'): void
}>()
</script>

<style scoped>
.workspace-resource-hover-card {
  --resource-hover-surface: var(--bg-primary, var(--el-bg-color, #f3f6fb));
  --resource-hover-surface-soft: var(--bg-secondary, var(--el-fill-color-light, #edf2f8));
  --resource-hover-surface-muted: var(--bg-tertiary, var(--el-fill-color, #e2e8f0));
  --resource-hover-border: var(--border-light, var(--el-border-color-light, rgba(71, 85, 105, 0.14)));
  --resource-hover-border-strong: var(--border-base, var(--el-border-color, rgba(71, 85, 105, 0.22)));
  --resource-hover-shadow: var(--app-shell-panel-shadow, var(--app-panel-shadow, 0 18px 44px rgba(15, 23, 42, 0.1)));
  position: fixed;
  z-index: 4200;
  width: min(340px, calc(100vw - 24px));
  padding: 12px;
  box-sizing: border-box;
  border: 1px solid var(--resource-hover-border);
  border-radius: var(--border-radius-lg, 8px);
  background: var(--resource-hover-surface);
  box-shadow:
    var(--resource-hover-shadow),
    inset 0 1px 0 var(--app-shell-panel-highlight, rgba(255, 255, 255, 0.72));
  color: var(--text-primary, var(--el-text-color-primary, #172033));
  pointer-events: auto;
  backdrop-filter: var(--app-shell-panel-backdrop, blur(12px) saturate(1.02));
}

.workspace-resource-hover-card__head {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.workspace-resource-hover-card__icon {
  width: 34px;
  height: 34px;
  flex: 0 0 34px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: var(--border-radius-lg, 8px);
  background: var(--resource-hover-surface-soft);
  border: 1px solid var(--resource-hover-border);
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.24);
}

.workspace-resource-hover-card__img,
.workspace-resource-hover-card__html-icon :deep(.workspace-resource-token__svg),
.workspace-resource-hover-card__html-icon :deep(.workspace-resource-token__glyph) {
  width: 20px;
  height: 20px;
  display: block;
}

.workspace-resource-hover-card__html-icon :deep(.workspace-resource-token__glyph) {
  position: relative;
  border-radius: 6px;
  color: transparent;
  background: rgba(var(--color-primary-rgb, 79, 70, 229), 0.1);
  border: 1px solid rgba(var(--color-primary-rgb, 79, 70, 229), 0.2);
}

.workspace-resource-hover-card__html-icon :deep(.workspace-resource-token__glyph::after) {
  content: "{}";
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--color-primary, #4f46e5);
  font-size: 10px;
  font-weight: 800;
  letter-spacing: 0;
}

.workspace-resource-hover-card__title-wrap {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.workspace-resource-hover-card__title {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--text-primary, var(--el-text-color-primary, #172033));
  font-size: 14px;
  font-weight: 650;
}

.workspace-resource-hover-card__type {
  color: var(--text-secondary, var(--el-text-color-secondary, #64748b));
  font-size: 12px;
  line-height: 1.3;
}

.workspace-resource-hover-card__close {
  width: 24px;
  height: 24px;
  border: 0;
  border-radius: 6px;
  background: transparent;
  color: var(--text-secondary, var(--el-text-color-secondary, #64748b));
  font-size: 18px;
  line-height: 1;
  cursor: pointer;
}

.workspace-resource-hover-card__close:hover {
  background: var(--resource-hover-surface-soft);
  color: var(--text-primary, var(--el-text-color-primary, #172033));
}

.workspace-resource-hover-card__description {
  margin: 9px 0 0;
  color: var(--text-regular, var(--el-text-color-regular, #334155));
  font-size: 12px;
  line-height: 1.55;
  display: -webkit-box;
  -webkit-line-clamp: 3;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.workspace-resource-hover-card__meta {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 10px;
}

.workspace-resource-hover-card__meta-item {
  max-width: 100%;
  padding: 3px 7px;
  border-radius: 7px;
  border: 1px solid var(--resource-hover-border);
  background: var(--resource-hover-surface-soft);
  color: var(--text-regular, var(--el-text-color-regular, #334155));
  font-size: 11px;
  line-height: 1.3;
  overflow-wrap: anywhere;
}

.workspace-resource-hover-card__path {
  display: block;
  margin-top: 10px;
  padding: 7px 8px;
  border-radius: 7px;
  border: 1px solid var(--app-code-border, rgba(51, 65, 85, 0.72));
  background: var(--app-code-bg, #1e293b);
  color: var(--app-code-text, #f8fafc);
  font-size: 11px;
  line-height: 1.45;
  white-space: normal;
  overflow-wrap: anywhere;
}

.workspace-resource-hover-card__loading {
  margin-top: 8px;
  color: var(--text-secondary, var(--el-text-color-secondary, #64748b));
  font-size: 12px;
}

.workspace-resource-hover-card__actions {
  display: flex;
  justify-content: flex-end;
  margin-top: 10px;
}

.workspace-resource-hover-card__open {
  height: 28px;
  padding: 0 10px;
  border: 1px solid var(--resource-hover-border-strong);
  border-radius: 7px;
  background: var(--resource-hover-surface-soft);
  color: var(--text-primary, var(--el-text-color-primary, #172033));
  font-size: 12px;
  cursor: pointer;
}

.workspace-resource-hover-card__open:hover {
  border-color: var(--color-primary, #4f46e5);
  background: rgba(var(--color-primary-rgb, 79, 70, 229), 0.1);
  color: var(--color-primary, #4f46e5);
}

:global(html.dark) .workspace-resource-hover-card {
  --resource-hover-surface: var(--bg-secondary, #1e293b);
  --resource-hover-surface-soft: var(--bg-tertiary, #334155);
  --resource-hover-surface-muted: var(--bg-primary, #0f172a);
  --resource-hover-border: var(--border-light, #334155);
  --resource-hover-border-strong: var(--border-base, #475569);
}

:global(html.dark) .workspace-resource-hover-card__icon {
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.08);
}

.workspace-resource-hover-card-fade-enter-active,
.workspace-resource-hover-card-fade-leave-active {
  transition: opacity 0.14s ease, transform 0.14s ease;
}

.workspace-resource-hover-card-fade-enter-from,
.workspace-resource-hover-card-fade-leave-to {
  opacity: 0;
  transform: translateY(-3px);
}
</style>
