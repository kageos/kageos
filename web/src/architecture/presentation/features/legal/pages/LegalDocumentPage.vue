<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { ArrowLeft } from '@element-plus/icons-vue'
import { getLegalDocument, type LegalDocumentKind } from '../legalDocuments'

const route = useRoute()
const router = useRouter()
const { locale } = useI18n()

const kind = computed<LegalDocumentKind>(() => route.meta.legalDocument === 'privacy' ? 'privacy' : 'terms')
const document = computed(() => getLegalDocument(kind.value, locale.value))

function goBack() {
  if (window.history.length > 1) {
    router.back()
    return
  }
  router.push('/login')
}
</script>

<template>
  <main class="legal-page" data-testid="legal-document-page">
    <header class="legal-header">
      <button type="button" class="back-button" @click="goBack">
        <el-icon><ArrowLeft /></el-icon>
        <span>{{ locale.toLowerCase().startsWith('zh') ? '返回' : 'Back' }}</span>
      </button>
      <RouterLink to="/login" class="brand-link">kageos</RouterLink>
    </header>

    <article class="legal-document">
      <div class="document-heading">
        <p class="document-kicker">LEGAL · {{ kind === 'privacy' ? 'PRIVACY' : 'TERMS' }}</p>
        <h1>{{ document.title }}</h1>
        <p class="document-summary">{{ document.summary }}</p>
        <p class="effective-date">
          {{ locale.toLowerCase().startsWith('zh') ? '生效日期' : 'Effective date' }}：{{ document.effectiveDate }}
        </p>
      </div>

      <nav class="document-switch" :aria-label="locale.toLowerCase().startsWith('zh') ? '法律文件' : 'Legal documents'">
        <RouterLink to="/legal/terms" :class="{ active: kind === 'terms' }">
          {{ locale.toLowerCase().startsWith('zh') ? '服务协议' : 'Terms of Service' }}
        </RouterLink>
        <RouterLink to="/legal/privacy" :class="{ active: kind === 'privacy' }">
          {{ locale.toLowerCase().startsWith('zh') ? '隐私政策' : 'Privacy Policy' }}
        </RouterLink>
      </nav>

      <section v-for="section in document.sections" :key="section.title" class="legal-section">
        <h2>{{ section.title }}</h2>
        <p v-for="paragraph in section.paragraphs" :key="paragraph">{{ paragraph }}</p>
        <ul v-if="section.bullets?.length">
          <li v-for="item in section.bullets" :key="item">{{ item }}</li>
        </ul>
      </section>
    </article>
  </main>
</template>

<style scoped>
.legal-page {
  min-height: 100vh;
  color: #172033;
  background:
    radial-gradient(circle at 12% 0%, rgba(37, 99, 235, 0.08), transparent 30%),
    #f5f7fb;
}

.legal-header {
  position: sticky;
  top: 0;
  z-index: 2;
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 64px;
  padding: 0 28px;
  border-bottom: 1px solid rgba(148, 163, 184, 0.22);
  background: rgba(255, 255, 255, 0.9);
  backdrop-filter: blur(16px);
}

.back-button {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  border: 0;
  color: #475569;
  background: transparent;
  cursor: pointer;
}

.brand-link {
  color: #0f2a54;
  font-size: 20px;
  font-weight: 800;
  text-decoration: none;
}

.legal-document {
  width: min(100% - 32px, 880px);
  margin: 40px auto 72px;
  padding: 48px 56px;
  border: 1px solid rgba(148, 163, 184, 0.22);
  border-radius: 18px;
  background: #fff;
  box-shadow: 0 20px 60px rgba(37, 62, 97, 0.08);
}

.document-heading {
  padding-bottom: 28px;
  border-bottom: 1px solid #e8edf4;
}

.document-kicker {
  margin: 0 0 12px;
  color: #2563eb;
  font-size: 12px;
  font-weight: 800;
  letter-spacing: 0.14em;
}

h1 {
  margin: 0;
  color: #10264a;
  font-size: clamp(30px, 5vw, 42px);
  line-height: 1.2;
}

.document-summary {
  margin: 16px 0 0;
  color: #526178;
  font-size: 16px;
  line-height: 1.8;
}

.effective-date {
  margin: 12px 0 0;
  color: #7b8799;
  font-size: 13px;
}

.document-switch {
  display: flex;
  gap: 10px;
  margin: 24px 0 34px;
}

.document-switch a {
  padding: 9px 14px;
  border: 1px solid #dce4ef;
  border-radius: 999px;
  color: #526178;
  font-size: 14px;
  text-decoration: none;
}

.document-switch a.active {
  border-color: rgba(37, 99, 235, 0.28);
  color: #1d4ed8;
  background: rgba(37, 99, 235, 0.08);
}

.legal-section + .legal-section {
  margin-top: 34px;
}

.legal-section h2 {
  margin: 0 0 13px;
  color: #172b4d;
  font-size: 19px;
  line-height: 1.45;
}

.legal-section p,
.legal-section li {
  color: #4d5b70;
  font-size: 15px;
  line-height: 1.9;
}

.legal-section p {
  margin: 0 0 10px;
}

.legal-section ul {
  margin: 8px 0 0;
  padding-left: 24px;
}

.legal-section li + li {
  margin-top: 5px;
}

@media (max-width: 640px) {
  .legal-header {
    height: 56px;
    padding: 0 16px;
  }

  .legal-document {
    width: 100%;
    margin: 0;
    padding: 32px 20px 56px;
    border: 0;
    border-radius: 0;
    box-shadow: none;
  }

  .document-switch {
    overflow-x: auto;
  }
}
</style>
