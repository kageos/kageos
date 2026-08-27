<template>
  <div class="system-settings-page">
    <el-card shadow="hover" class="settings-card">
      <template #header>
        <div class="card-header">
          <div>
            <h2>{{ t('route.systemSettings') }}</h2>
            <p>{{ t('systemSettings.subtitle') }}</p>
          </div>
          <div class="header-actions">
            <el-button :icon="Refresh" @click="refreshActiveTab">{{ t('common.refresh') }}</el-button>
            <el-button v-if="activeTab === 'email'" type="primary" :icon="Check" :loading="saving" @click="saveSettings">
              {{ t('connectorProvider.save') }}
            </el-button>
          </div>
        </div>
      </template>

      <div class="settings-layout">
        <aside class="settings-sidebar" :aria-label="t('systemSettings.categoryLabel')">
          <button
            v-for="section in settingsSections"
            :key="section.key"
            type="button"
            class="settings-nav-item"
            :class="{ 'is-active': activeTab === section.key }"
            @click="selectSettingsSection(section.key)"
          >
            <span class="settings-nav-title">{{ section.title }}</span>
            <span class="settings-nav-desc">{{ section.desc }}</span>
          </button>
        </aside>

        <section class="settings-content">
          <div class="section-header">
            <div>
              <h3>{{ currentSection.title }}</h3>
              <p>{{ currentSection.desc }}</p>
            </div>
            <el-button :icon="QuestionFilled" @click="openCurrentDocs">
              {{ t('systemSettings.viewDocs') }}
            </el-button>
          </div>

          <div v-if="activeTab === 'operations'" v-loading="resourcesLoading" class="section-pane operations-pane">
            <template v-if="resourceOverview">
              <el-alert
                :title="forecastTitle"
                :description="forecastDescription"
                :type="forecastAlertType"
                show-icon
                :closable="false"
              />
              <el-alert
                :title="environmentTitle"
                :description="environmentDescription"
                type="info"
                show-icon
                :closable="false"
              />

              <div class="resource-summary-grid">
                <article class="resource-summary-card">
                  <span class="resource-summary-label">{{ t('systemSettings.resources.disk') }}</span>
                  <strong>{{ formatBytes(resourceOverview.current.disk_used_bytes) }} / {{ formatBytes(resourceOverview.current.disk_total_bytes) }}</strong>
                  <el-progress :percentage="roundedPercent(resourceOverview.current.disk_used_percent)" :status="progressStatus(resourceOverview.current.disk_used_percent)" />
                  <small>{{ t('systemSettings.resources.diskFree', { value: formatBytes(resourceOverview.current.disk_free_bytes) }) }}</small>
                </article>
                <article class="resource-summary-card">
                  <span class="resource-summary-label">{{ t('systemSettings.resources.memory') }}</span>
                  <strong v-if="resourceOverview.current.memory_available">
                    {{ formatBytes(resourceOverview.current.memory_used_bytes) }} / {{ formatBytes(resourceOverview.current.memory_total_bytes) }}
                  </strong>
                  <strong v-else>{{ t('systemSettings.resources.unavailable') }}</strong>
                  <el-progress v-if="resourceOverview.current.memory_available" :percentage="roundedPercent(resourceOverview.current.memory_used_percent)" />
                  <small>{{ t('systemSettings.resources.cpuCores', { count: resourceOverview.current.cpu_cores }) }}</small>
                </article>
                <article class="resource-summary-card">
                  <span class="resource-summary-label">{{ t('systemSettings.resources.cpuAndLoad') }}</span>
                  <strong v-if="resourceOverview.current.cpu_available">
                    {{ t('systemSettings.resources.cpuUsageValue', { value: roundedPercent(resourceOverview.current.cpu_used_percent) }) }}
                  </strong>
                  <strong v-else>{{ t('systemSettings.resources.unavailable') }}</strong>
                  <small v-if="resourceOverview.current.load_available">{{ t('systemSettings.resources.loadCurrent', { one: resourceOverview.current.load_1.toFixed(2), five: resourceOverview.current.load_5.toFixed(2), fifteen: resourceOverview.current.load_15.toFixed(2) }) }}</small>
                  <small v-else>{{ t('systemSettings.resources.cpuUsageHint') }}</small>
                </article>
                <article class="resource-summary-card">
                  <span class="resource-summary-label">{{ t('systemSettings.resources.host') }}</span>
                  <strong>{{ resourceOverview.current.hostname || '-' }}</strong>
                  <small>{{ resourceOverview.current.operating_system }} / {{ resourceOverview.current.architecture }} · {{ formatUptime(resourceOverview.current.uptime_seconds) }}</small>
                </article>
                <article class="resource-summary-card">
                  <span class="resource-summary-label">{{ t('systemSettings.resources.network') }}</span>
                  <strong v-if="resourceOverview.current.network_available">↓ {{ formatRate(resourceOverview.current.network_rx_bytes_per_second) }} · ↑ {{ formatRate(resourceOverview.current.network_tx_bytes_per_second) }}</strong>
                  <strong v-else>{{ t('systemSettings.resources.unavailable') }}</strong>
                  <small v-if="resourceOverview.current.network_available">{{ t('systemSettings.resources.networkTotal', { rx: formatBytes(resourceOverview.current.network_rx_bytes), tx: formatBytes(resourceOverview.current.network_tx_bytes) }) }}</small>
                </article>
                <article class="resource-summary-card">
                  <span class="resource-summary-label">{{ t('systemSettings.resources.diskIO') }}</span>
                  <strong v-if="resourceOverview.current.disk_io_available">↓ {{ formatRate(resourceOverview.current.disk_read_bytes_per_second) }} · ↑ {{ formatRate(resourceOverview.current.disk_write_bytes_per_second) }}</strong>
                  <strong v-else>{{ t('systemSettings.resources.unavailable') }}</strong>
                  <small v-if="resourceOverview.current.disk_io_available">{{ t('systemSettings.resources.diskIOTotal', { read: formatBytes(resourceOverview.current.disk_read_bytes), write: formatBytes(resourceOverview.current.disk_write_bytes) }) }}</small>
                </article>
                <article class="resource-summary-card">
                  <span class="resource-summary-label">{{ t('systemSettings.resources.swap') }}</span>
                  <strong v-if="resourceOverview.current.swap_total_bytes > 0">{{ formatBytes(resourceOverview.current.swap_used_bytes) }} / {{ formatBytes(resourceOverview.current.swap_total_bytes) }}</strong>
                  <strong v-else>{{ t('systemSettings.resources.notConfigured') }}</strong>
                  <el-progress v-if="resourceOverview.current.swap_total_bytes > 0" :percentage="roundedPercent(resourceOverview.current.swap_used_percent)" />
                  <small>{{ t('systemSettings.resources.runtimeFrequency', { seconds: resourceOverview.runtime_interval_seconds, minutes: resourceOverview.sample_interval_minutes }) }}</small>
                </article>
              </div>

              <div class="resource-panel">
                <div class="resource-panel-heading">
                  <div>
                    <h4>{{ t('systemSettings.resources.storagePoolsTitle') }}</h4>
                    <p>{{ t('systemSettings.resources.storagePoolsDesc') }}</p>
                  </div>
                </div>
                <div class="storage-pool-grid">
                  <article v-for="pool in resourceOverview.current.storage_pools" :key="pool.key" class="storage-pool-card">
                    <div class="storage-pool-title">
                      <strong>{{ storagePoolName(pool.key, pool.name) }}</strong>
                      <el-tag v-if="pool.primary" size="small" effect="plain">{{ t('systemSettings.resources.primaryPool') }}</el-tag>
                    </div>
                    <div v-if="pool.available" class="storage-pool-amount">{{ formatBytes(pool.used_bytes) }} / {{ formatBytes(pool.total_bytes) }}</div>
                    <div v-else class="storage-pool-amount">{{ t('systemSettings.resources.usedKnownCapacityUnknown', { value: formatBytes(pool.used_bytes) }) }}</div>
                    <el-progress v-if="pool.available" :percentage="roundedPercent(pool.used_percent)" :status="progressStatus(pool.used_percent)" />
                    <small v-if="pool.available">{{ t('systemSettings.resources.diskFree', { value: formatBytes(pool.free_bytes) }) }}</small>
                    <small v-else>{{ t('systemSettings.resources.capacityUnavailableHint') }}</small>
                  </article>
                </div>
              </div>

              <div class="resource-panel">
                <div class="resource-panel-heading">
                  <div>
                    <h4>{{ t('systemSettings.resources.platformTitle') }}</h4>
                    <p>{{ t('systemSettings.resources.platformDesc', { time: formatResourceTime(resourceOverview.platform.collected_at) }) }}</p>
                  </div>
                </div>
                <div class="platform-metric-grid">
                  <article class="platform-metric-card"><span>{{ t('systemSettings.resources.users') }}</span><strong>{{ resourceOverview.platform.users_total }}</strong><small>{{ t('systemSettings.resources.activeAndPending', { active: resourceOverview.platform.users_active, pending: resourceOverview.platform.users_pending }) }}</small></article>
                  <article class="platform-metric-card"><span>{{ t('systemSettings.resources.workspaces') }}</span><strong>{{ resourceOverview.platform.app_stats_available ? resourceOverview.platform.workspaces_total : '-' }}</strong><small>{{ resourceOverview.platform.app_stats_available ? t('systemSettings.resources.enabledCount', { count: resourceOverview.platform.workspaces_enabled }) : t('systemSettings.resources.sourceUnavailable') }}</small></article>
                  <article class="platform-metric-card"><span>{{ t('systemSettings.resources.serviceDirectories') }}</span><strong>{{ resourceOverview.platform.app_stats_available ? resourceOverview.platform.service_directories : '-' }}</strong><small>{{ resourceOverview.platform.app_stats_available ? t('systemSettings.resources.functionsCount', { count: resourceOverview.platform.functions_total }) : t('systemSettings.resources.sourceUnavailable') }}</small></article>
                  <article class="platform-metric-card"><span>{{ t('systemSettings.resources.appDatabases') }}</span><strong>{{ resourceOverview.platform.runtime_stats_available ? resourceOverview.platform.app_databases_total : '-' }}</strong><small>{{ resourceOverview.current.database_size_available ? t('systemSettings.resources.logicalDatabaseSize', { value: formatBytes(resourceOverview.current.database_logical_bytes) }) : t('systemSettings.resources.databaseSizeUnavailable') }}</small></article>
                  <article class="platform-metric-card"><span>{{ t('systemSettings.resources.scheduledTasks') }}</span><strong>{{ resourceOverview.platform.timer_stats_available ? resourceOverview.platform.scheduled_tasks_total : '-' }}</strong><small>{{ resourceOverview.platform.timer_stats_available ? t('systemSettings.resources.activeTasks', { count: resourceOverview.platform.scheduled_tasks_active }) : t('systemSettings.resources.sourceUnavailable') }}</small></article>
                </div>
                <el-table v-if="resourceOverview.current.largest_databases.length" :data="resourceOverview.current.largest_databases" size="small" class="database-size-table">
                  <el-table-column prop="name" :label="t('systemSettings.resources.databaseName')" min-width="200" />
                  <el-table-column :label="t('systemSettings.resources.logicalSize')" min-width="140"><template #default="{ row }">{{ formatBytes(row.used_bytes) }}</template></el-table-column>
                </el-table>
              </div>

              <div class="resource-panel">
                <div class="resource-panel-heading">
                  <div>
                    <h4>{{ t('systemSettings.resources.historyTitle') }}</h4>
                    <p>{{ t('systemSettings.resources.historyDesc', { minutes: resourceOverview.sample_interval_minutes }) }}</p>
                  </div>
                  <el-radio-group v-model="resourceHistoryHours" size="small" @change="loadResources">
                    <el-radio-button :value="24">24h</el-radio-button>
                    <el-radio-button :value="168">7d</el-radio-button>
                    <el-radio-button :value="720">30d</el-radio-button>
                  </el-radio-group>
                </div>
                <div v-if="resourceOverview.history.length > 1" class="resource-chart-wrap">
                  <svg class="resource-chart" viewBox="0 0 800 180" preserveAspectRatio="none" role="img" :aria-label="t('systemSettings.resources.historyTitle')">
                    <line v-for="level in [25, 50, 75]" :key="level" x1="0" :y1="180 - level * 1.8" x2="800" :y2="180 - level * 1.8" class="chart-grid-line" />
                    <polyline :points="historyPolyline('disk_used_percent')" class="chart-line disk-line" />
                    <polyline :points="historyPolyline('memory_used_percent')" class="chart-line memory-line" />
                    <polyline :points="historyPolyline('cpu_used_percent')" class="chart-line cpu-line" />
                  </svg>
                  <div class="chart-legend">
                    <span><i class="legend-dot disk-dot" />{{ t('systemSettings.resources.diskUsage') }}</span>
                    <span><i class="legend-dot memory-dot" />{{ t('systemSettings.resources.memoryUsage') }}</span>
                    <span><i class="legend-dot cpu-dot" />{{ t('systemSettings.resources.cpuUsage') }}</span>
                    <span>{{ historyTimeRange }}</span>
                  </div>
                  <svg class="resource-chart rate-chart" viewBox="0 0 800 120" preserveAspectRatio="none" role="img" :aria-label="t('systemSettings.resources.networkTrend')">
                    <polyline :points="historyRatePolyline('network_rx_bytes_per_second')" class="chart-line network-rx-line" />
                    <polyline :points="historyRatePolyline('network_tx_bytes_per_second')" class="chart-line network-tx-line" />
                  </svg>
                  <div class="chart-legend">
                    <span><i class="legend-dot network-rx-dot" />{{ t('systemSettings.resources.networkDownload') }}</span>
                    <span><i class="legend-dot network-tx-dot" />{{ t('systemSettings.resources.networkUpload') }}</span>
                    <span>{{ t('systemSettings.resources.networkTrendScale') }}</span>
                  </div>
                </div>
                <el-empty v-else :description="t('systemSettings.resources.historyCollecting')" />
              </div>
              <div class="resource-panel">
                <div class="resource-panel-heading"><div><h4>{{ t('systemSettings.resources.collectionTitle') }}</h4><p>{{ t('systemSettings.resources.collectionDesc') }}</p></div></div>
                <el-table :data="resourceOverview.collection_tasks" size="small">
                  <el-table-column :label="t('systemSettings.resources.collectionTask')" min-width="150"><template #default="{ row }">{{ collectionTaskName(row.key) }}</template></el-table-column>
                  <el-table-column :label="t('systemSettings.resources.collectionStatus')" width="110"><template #default="{ row }"><el-tag :type="collectionStatusType(row.status)" size="small">{{ collectionStatusLabel(row.status) }}</el-tag></template></el-table-column>
                  <el-table-column :label="t('systemSettings.resources.lastSuccess')" min-width="180"><template #default="{ row }">{{ row.last_succeeded_at ? formatResourceTime(row.last_succeeded_at) : '-' }}</template></el-table-column>
                  <el-table-column :label="t('systemSettings.resources.nextCollection')" min-width="180"><template #default="{ row }">{{ row.next_run_at ? formatResourceTime(row.next_run_at) : '-' }}</template></el-table-column>
                  <el-table-column :label="t('systemSettings.resources.collectionDuration')" width="120"><template #default="{ row }">{{ formatDurationMillis(row.duration_millis) }}</template></el-table-column>
                  <el-table-column :label="t('systemSettings.resources.collectionResult')" min-width="180"><template #default="{ row }"><span :class="{ 'collection-error': row.error }">{{ collectionResult(row.error) }}</span></template></el-table-column>
                </el-table>
              </div>

              <div class="resource-panel">
                <div class="resource-panel-heading">
                  <div>
                    <h4>{{ t('systemSettings.resources.breakdownTitle') }}</h4>
                    <p>{{ t('systemSettings.resources.breakdownDesc') }}</p>
                  </div>
                </div>
                <el-table :data="resourceOverview.current.components" stripe>
                  <el-table-column :label="t('systemSettings.resources.service')" min-width="180">
                    <template #default="{ row }">{{ resourceComponentName(row.key, row.name) }}</template>
                  </el-table-column>
                  <el-table-column :label="t('systemSettings.resources.storagePool')" min-width="150">
                    <template #default="{ row }">{{ storagePoolName(row.pool_key, row.pool_key) }}</template>
                  </el-table-column>
                  <el-table-column :label="t('systemSettings.resources.usedSpace')" min-width="160">
                    <template #default="{ row }">{{ row.available ? formatBytes(row.used_bytes) : t('systemSettings.resources.notMounted') }}</template>
                  </el-table-column>
                  <el-table-column :label="t('systemSettings.resources.share')" min-width="220">
                    <template #default="{ row }">
                      <el-progress v-if="row.available" :percentage="componentDiskPercent(row.used_bytes, row.pool_key)" />
                      <span v-else>-</span>
                    </template>
                  </el-table-column>
                </el-table>
              </div>
              <p class="resource-collected-at">{{ t('systemSettings.resources.collectedAt', { time: formatResourceTime(resourceOverview.current.collected_at) }) }}</p>
            </template>
            <el-empty v-else-if="!resourcesLoading" :description="t('systemSettings.resources.empty')" />
          </div>

          <div v-else-if="activeTab === 'email'" v-loading="loading" class="section-pane">
            <el-alert
              v-if="form.registration_mode === 'admin_only'"
              :title="t('systemSettings.registrationDisabled')"
              type="info"
              show-icon
              :closable="false"
            />

            <el-form ref="formRef" :model="form" label-width="120px" class="settings-form">
              <el-divider content-position="left">{{ t('systemSettings.registration') }}</el-divider>
              <el-form-item :label="t('systemSettings.registrationMode')">
                <el-radio-group v-model="form.registration_mode">
                  <el-radio-button value="admin_only">{{ t('systemSettings.adminOnly') }}</el-radio-button>
                  <el-radio-button value="email_code">{{ t('systemSettings.emailVerification') }}</el-radio-button>
                  <el-radio-button value="debug_code">{{ t('systemSettings.debugCode') }}</el-radio-button>
                </el-radio-group>
              </el-form-item>

              <el-divider content-position="left">{{ t('systemSettings.emailService') }}</el-divider>
              <el-form-item :label="t('systemSettings.emailMode')">
                <el-radio-group v-model="form.email.mode">
                  <el-radio-button value="smtp">SMTP</el-radio-button>
                  <el-radio-button value="log">Log</el-radio-button>
                </el-radio-group>
              </el-form-item>

              <el-form-item :label="t('systemSettings.smtpHost')">
                <el-input v-model="form.email.host" placeholder="smtp.example.com" />
              </el-form-item>
              <el-form-item :label="t('systemSettings.smtpPort')">
                <el-input-number v-model="form.email.port" :min="1" :max="65535" />
              </el-form-item>
              <el-form-item :label="t('systemSettings.username')">
                <el-input v-model="form.email.username" :placeholder="t('systemSettings.smtpUsernamePlaceholder')" />
              </el-form-item>
              <el-form-item :label="t('systemSettings.password')">
                <el-input
                  v-model="form.email.password"
                  type="password"
                  show-password
                  :placeholder="form.email.password_set ? t('systemSettings.smtpPasswordKeepPlaceholder') : t('systemSettings.smtpPasswordPlaceholder')"
                />
              </el-form-item>
              <el-form-item :label="t('systemSettings.from')">
                <el-input v-model="form.email.from" placeholder="noreply@example.com" />
              </el-form-item>
              <el-form-item :label="t('systemSettings.fromName')">
                <el-input v-model="form.email.from_name" placeholder="kageos" />
              </el-form-item>

              <el-divider content-position="left">{{ t('systemSettings.testEmail') }}</el-divider>
              <el-form-item :label="t('systemSettings.recipient')">
                <div class="test-row">
                  <el-input v-model="testEmail" placeholder="admin@example.com" />
                  <el-button :icon="Message" :loading="testing" @click="sendTestEmail">
                    {{ t('systemSettings.sendTest') }}
                  </el-button>
                </div>
              </el-form-item>
            </el-form>
          </div>

          <div v-else-if="activeTab === 'login'" class="section-pane">
            <div v-loading="providersLoading" class="login-provider-section">
              <div class="login-announcement-panel">
                <div class="login-announcement-heading">
                  <div>
                    <h4>{{ t('systemSettings.loginAnnouncementTitle') }}</h4>
                    <p>{{ t('systemSettings.loginAnnouncementDesc') }}</p>
                  </div>
                  <el-switch
                    v-model="loginAnnouncement.enabled"
                    :active-text="t('systemSettings.on')"
                    :inactive-text="t('systemSettings.off')"
                  />
                </div>
                <el-form label-width="96px">
                  <el-form-item :label="t('systemSettings.loginAnnouncementMarkdown')">
                    <el-input
                      v-model="loginAnnouncement.markdown"
                      type="textarea"
                      :rows="8"
                      maxlength="10000"
                      show-word-limit
                      :placeholder="t('systemSettings.loginAnnouncementMarkdownPlaceholder')"
                    />
                  </el-form-item>
                  <div class="login-announcement-actions">
                    <el-button type="primary" :icon="Check" :loading="announcementSaving" @click="saveLoginAnnouncement">
                      {{ t('systemSettings.saveLoginAnnouncement') }}
                    </el-button>
                  </div>
                </el-form>
              </div>

              <div class="provider-summary">
                <div class="summary-item">
                  <span class="summary-value">{{ authProviders.length }}</span>
                  <span class="summary-label">{{ t('systemSettings.loginPresetCount') }}</span>
                </div>
                <div class="summary-item">
                  <span class="summary-value">{{ configuredProviderCount }}</span>
                  <span class="summary-label">{{ t('systemSettings.configuredCount') }}</span>
                </div>
                <div class="summary-item">
                  <span class="summary-value">{{ enabledProviderCount }}</span>
                  <span class="summary-label">{{ t('systemSettings.enabledCount') }}</span>
                </div>
              </div>

              <el-empty v-if="!authProviders.length && !providersLoading" :description="t('systemSettings.noLoginProviders')" />

              <div
                v-for="provider in authProviders"
                :key="provider.code"
                class="provider-panel"
              >
                <div class="provider-panel-header">
                  <div class="provider-heading">
                    <div class="provider-title-row">
                      <span class="provider-name">{{ provider.name }}</span>
                      <el-tag :type="providerStatusType(provider)" size="small">
                        {{ providerStatusLabel(provider) }}
                      </el-tag>
                      <el-tag size="small" effect="plain">
                        {{ providerActionLabel(provider.action) }}
                      </el-tag>
                    </div>
                    <p>{{ provider.description }}</p>
                    <div class="provider-meta">
                      <span v-if="provider.callback_path" class="callback-path">
                        {{ t('systemSettings.callbackUrl') }}：<code>{{ callbackURL(provider) }}</code>
                      </span>
                      <el-button
                        v-if="provider.callback_path"
                        link
                        type="primary"
                        :icon="CopyDocument"
                        @click="copyCallbackURL(provider)"
                      >
                        {{ t('connectorProvider.copy') }}
                      </el-button>
                      <el-link
                        v-if="provider.docs_url"
                        type="primary"
                        :href="provider.docs_url"
                        target="_blank"
                        :underline="false"
                      >
                        {{ t('systemSettings.platformDocs') }}
                      </el-link>
                    </div>
                  </div>
                  <div class="provider-enable">
                    <span>{{ t('systemSettings.enabled') }}</span>
                    <el-switch
                      :model-value="provider.enabled"
                      :disabled="!provider.configured"
                      :loading="providerSwitching[provider.code]"
                      @change="handleProviderSwitchChange(provider, $event)"
                    />
                  </div>
                </div>

                <el-form
                  v-if="providerConfigs[provider.code]"
                  :model="providerConfigs[provider.code]"
                  label-width="120px"
                  class="provider-form"
                >
                  <el-form-item
                    v-for="field in provider.fields"
                    :key="field.key"
                    :label="field.label"
                    :required="field.required"
                  >
                    <div class="provider-field">
                      <el-switch
                        v-if="field.type === 'boolean'"
                        :model-value="providerBooleanFieldValue(provider, field)"
                        :active-text="t('systemSettings.on')"
                        :inactive-text="t('systemSettings.off')"
                        @change="setProviderFieldValue(provider, field, $event)"
                      />
                      <el-input
                        v-else
                        :model-value="providerFieldValue(provider, field)"
                        :type="fieldInputType(field)"
                        :show-password="field.secret"
                        :placeholder="fieldPlaceholder(field)"
                        @update:model-value="setProviderFieldValue(provider, field, $event)"
                      >
                        <template v-if="isCallbackField(field)" #append>
                          <el-button @click="fillCallbackURL(provider, field.key)">
                            {{ t('systemSettings.useCurrentCallback') }}
                          </el-button>
                        </template>
                      </el-input>
                      <div v-if="field.help" class="field-help">{{ field.help }}</div>
                    </div>
                  </el-form-item>
                  <el-form-item>
                    <div class="provider-form-actions">
                      <el-button
                        type="primary"
                        :icon="Check"
                        :loading="providerSaving[provider.code]"
                        @click="saveProviderConfig(provider)"
                      >
                        {{ t('systemSettings.saveProvider') }}
                      </el-button>
                      <el-button
                        :disabled="!provider.configured"
                        :loading="providerSwitching[provider.code]"
                        @click="handleProviderEnabledChange(provider, !provider.enabled)"
                      >
                        {{ provider.enabled ? t('systemSettings.disable') : t('systemSettings.enable') }}
                      </el-button>
                    </div>
                  </el-form-item>
                </el-form>
              </div>
            </div>
          </div>

          <div v-else-if="activeTab === 'connectors'" class="section-pane">
            <ConnectorProviderManagementPage :key="connectorPanelKey" embedded />
          </div>

          <div v-else-if="activeTab === 'openapi'" class="section-pane">
            <OpenAPITokenManagementPage :key="openapiPanelKey" embedded />
          </div>

          <div v-else-if="activeTab === 'users'" class="section-pane">
            <SystemUserManagementPage :key="usersPanelKey" />
          </div>

          <div v-else-if="activeTab === 'backups'" v-loading="archivesLoading" class="section-pane">
            <el-alert :title="t('systemSettings.archiveFallbackNote')" type="info" show-icon :closable="false" />
            <div class="archive-summary-row">
              <span>{{ t('systemSettings.archiveRetention', { days: archiveRetentionDays }) }}</span>
              <span>{{ t('systemSettings.archiveSchedule', { cron: archiveCronExpr, timezone: archiveTimezone }) }}</span>
            </div>
            <el-table :data="archiveBatches" stripe class="archive-table">
              <el-table-column :label="t('systemSettings.archiveScope')" min-width="150">
                <template #default="{ row }">{{ row.tenant_user }}/{{ row.app }}</template>
              </el-table-column>
              <el-table-column :label="t('systemSettings.archiveRange')" min-width="220">
                <template #default="{ row }">{{ formatArchiveTime(row.range_started_at) }} — {{ formatArchiveTime(row.range_ended_at) }}</template>
              </el-table-column>
              <el-table-column prop="record_count" :label="t('systemSettings.archiveRecords')" width="110" />
              <el-table-column :label="t('systemSettings.archiveSize')" width="110">
                <template #default="{ row }">{{ formatFileSize(row.file_size) }}</template>
              </el-table-column>
              <el-table-column :label="t('systemSettings.archiveStatus')" width="110">
                <template #default="{ row }"><el-tag :type="archiveStatusType(row.status)" size="small">{{ archiveStatusLabel(row.status) }}</el-tag></template>
              </el-table-column>
              <el-table-column :label="t('systemSettings.archiveSummary')" min-width="240">
                <template #default="{ row }">{{ archiveResourceSummary(row) }}</template>
              </el-table-column>
              <el-table-column :label="t('systemSettings.archiveEvidence')" min-width="260">
                <template #default="{ row }">
                  <div class="archive-evidence"><code>{{ row.object_ref || '-' }}</code><code v-if="row.sha256">SHA256 {{ row.sha256 }}</code></div>
                </template>
              </el-table-column>
            </el-table>
            <el-empty v-if="!archiveBatches.length && !archivesLoading" :description="t('systemSettings.noArchives')" />
            <el-pagination
              v-if="archiveTotal > archivePageSize"
              v-model:current-page="archivePage"
              :page-size="archivePageSize"
              :total="archiveTotal"
              layout="prev, pager, next"
              @current-change="loadArchives"
            />
          </div>

          <div v-else-if="activeTab === 'appearance'" class="section-pane">
            <div class="preference-grid">
              <button
                v-for="theme in availableThemes"
                :key="theme.name"
                type="button"
                class="preference-card"
                :class="{ 'is-active': currentThemeName === theme.name }"
                @click="handleThemeChange(theme.name)"
              >
                <span class="preference-card-title">{{ theme.label }}</span>
                <span class="preference-card-desc">{{ theme.mode === 'dark' ? t('workspace.darkUi') : t('workspace.lightUi') }}</span>
              </button>
            </div>
          </div>

          <div v-else-if="activeTab === 'language'" class="section-pane">
            <div class="preference-grid">
              <button
                v-for="option in localeStore.localeOptions"
                :key="option.value"
                type="button"
                class="preference-card"
                :class="{ 'is-active': option.value === localeStore.currentLocale }"
                @click="handleLocaleChange(option.value)"
              >
                <span class="preference-card-title">{{ option.flag }} {{ option.nativeLabel }}</span>
                <span class="preference-card-desc">{{ option.englishLabel }}</span>
              </button>
            </div>
          </div>
        </section>
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { Check, CopyDocument, Message, QuestionFilled, Refresh } from '@element-plus/icons-vue'
import { useLocaleStore, useThemeStore } from '@/architecture/presentation/context/appStoresContext'
import type { SupportedLocale } from '@/architecture/shared/i18n'
import { getKageosDocsURL, openExternalURL, type KageosDocSlug } from '@/architecture/shared/config/externalLinks'
import { featureFlags } from '@/architecture/shared/config/features'
import ConnectorProviderManagementPage from '@/architecture/presentation/features/connector/pages/ConnectorProviderManagementPage.vue'
import OpenAPITokenManagementPage from '@/architecture/presentation/features/agent/pages/OpenAPITokenManagementPage.vue'
import SystemUserManagementPage from '@/architecture/presentation/features/system/pages/SystemUserManagementPage.vue'
import {
  getSystemSettings,
  getLoginAnnouncementConfig,
  getSystemResourceOverview,
  listAuthLoginProviders,
  listLogArchiveBatches,
  updateSystemSettings,
  updateLoginAnnouncementConfig,
  updateAuthLoginProviderConfig,
  updateAuthLoginProviderEnabled,
  testSystemEmail,
  type AuthLoginProviderField,
  type AuthLoginProviderInfo,
  type LogArchiveBatch,
  type LoginAnnouncement,
  type SystemResourceHistoryPoint,
  type SystemResourceOverview,
  type SystemSettings
} from '@/architecture/presentation/context/api/system-settings'

type SettingsTab = 'operations' | 'email' | 'login' | 'connectors' | 'openapi' | 'users' | 'backups' | 'appearance' | 'language'

interface SettingsSection {
  key: SettingsTab
  title: string
  desc: string
}

const loading = ref(false)
const saving = ref(false)
const testing = ref(false)
const testEmail = ref('')
const defaultSettingsTab: SettingsTab = 'operations'
const activeTab = ref<SettingsTab>(defaultSettingsTab)
const connectorPanelKey = ref(0)
const openapiPanelKey = ref(0)
const usersPanelKey = ref(0)
const archivesLoading = ref(false)
const archiveBatches = ref<LogArchiveBatch[]>([])
const archivePage = ref(1)
const archivePageSize = 20
const archiveTotal = ref(0)
const archiveRetentionDays = ref(90)
const archiveCronExpr = ref('20 3 * * *')
const archiveTimezone = ref('Asia/Shanghai')
const resourcesLoading = ref(false)
const resourceOverview = ref<SystemResourceOverview | null>(null)
const resourceHistoryHours = ref(168)
let resourceRefreshTimer: ReturnType<typeof setInterval> | undefined
let lastFullResourceRefreshAt = 0
const providersLoading = ref(false)
const announcementSaving = ref(false)
const authProviders = ref<AuthLoginProviderInfo[]>([])
const providerConfigs = reactive<Record<string, Record<string, string>>>({})
const providerSaving = reactive<Record<string, boolean>>({})
const providerSwitching = reactive<Record<string, boolean>>({})
const loginAnnouncement = reactive<LoginAnnouncement>({ enabled: false, markdown: '' })
const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const localeStore = useLocaleStore()
const themeStore = useThemeStore()

const allSettingsSections = computed<SettingsSection[]>(() => [
  { key: 'operations', title: t('systemSettings.sections.operationsTitle'), desc: t('systemSettings.sections.operationsDesc') },
  { key: 'email', title: t('systemSettings.sections.emailTitle'), desc: t('systemSettings.sections.emailDesc') },
  { key: 'login', title: t('systemSettings.sections.loginTitle'), desc: t('systemSettings.sections.loginDesc') },
  { key: 'users', title: t('systemSettings.sections.usersTitle'), desc: t('systemSettings.sections.usersDesc') },
  { key: 'backups', title: t('systemSettings.sections.backupsTitle'), desc: t('systemSettings.sections.backupsDesc') },
  { key: 'openapi', title: t('systemSettings.sections.openapiTitle'), desc: t('systemSettings.sections.openapiDesc') },
  { key: 'connectors', title: t('systemSettings.sections.connectorsTitle'), desc: t('systemSettings.sections.connectorsDesc') },
  { key: 'appearance', title: t('systemSettings.sections.appearanceTitle'), desc: t('systemSettings.sections.appearanceDesc') },
  { key: 'language', title: t('systemSettings.sections.languageTitle'), desc: t('systemSettings.sections.languageDesc') },
])

const settingsSections = computed<SettingsSection[]>(() => {
  return allSettingsSections.value.filter((section) => {
    if (section.key === 'email') return featureFlags.systemEmailSettings
    if (section.key === 'connectors') return featureFlags.connectorSettings
    if (section.key === 'openapi') return featureFlags.openapiTokens
    return true
  })
})

const settingsDocSlugMap: Record<SettingsTab, KageosDocSlug> = {
  operations: 'runtime',
  email: 'runtime',
  login: 'login',
  connectors: 'connectors',
  openapi: 'api',
  users: 'runtime',
  backups: 'runtime',
  appearance: 'docs',
  language: 'docs',
}

const currentSection = computed(() => {
  return settingsSections.value.find((section) => section.key === activeTab.value) || settingsSections.value[0]!
})

const currentDocsURL = computed(() => {
  return getKageosDocsURL(settingsDocSlugMap[activeTab.value], localeStore.currentLocale)
})

const availableThemes = computed(() => themeStore.getAvailableThemes())
const currentThemeName = computed(() => themeStore.currentTheme.name)

const forecastAlertType = computed(() => {
  if (resourceOverview.value?.forecast.status === 'critical') return 'error'
  if (resourceOverview.value?.forecast.status === 'warning') return 'warning'
  return 'success'
})

const forecastTitle = computed(() => {
  const status = resourceOverview.value?.forecast.status || 'healthy'
  return t(`systemSettings.resources.forecast.${status}`)
})

const forecastDescription = computed(() => {
  const forecast = resourceOverview.value?.forecast
  if (!forecast) return ''
  if (typeof forecast.days_to_target === 'number') {
    return t('systemSettings.resources.forecastDays', {
      days: forecast.days_to_target,
      target: forecast.target_percent,
      growth: formatBytes(forecast.daily_growth_bytes),
    })
  }
  if (forecast.status === 'critical') {
    return t('systemSettings.resources.forecastCriticalNow', {
      pool: storagePoolName(forecast.pool_key, forecast.pool_key),
      percent: roundedPercent(forecast.current_used_percent),
    })
  }
  if (forecast.status === 'warning') {
    return t('systemSettings.resources.forecastWarningNow', {
      pool: storagePoolName(forecast.pool_key, forecast.pool_key),
      percent: roundedPercent(forecast.current_used_percent),
      target: forecast.target_percent,
    })
  }
  return t('systemSettings.resources.forecastStable', { target: forecast.target_percent })
})

const historyTimeRange = computed(() => {
  const history = resourceOverview.value?.history || []
  if (!history.length) return ''
  return `${formatResourceTime(history[0].collected_at)} — ${formatResourceTime(history[history.length - 1].collected_at)}`
})

const environmentTitle = computed(() => {
  const environment = resourceOverview.value?.current.environment
  if (!environment) return ''
  return t(`systemSettings.resources.environments.${environment.mode}.${environment.deployment}`)
})

const environmentDescription = computed(() => {
  const environment = resourceOverview.value?.current.environment
  if (!environment) return ''
  const engine = environment.container_engine && environment.container_engine !== 'none'
    ? environment.container_engine
    : t('systemSettings.resources.noContainerEngine')
  return t('systemSettings.resources.environmentDesc', {
    os: resourceOverview.value?.current.operating_system,
    arch: resourceOverview.value?.current.architecture,
    engine,
    source: t(`systemSettings.resources.storageSources.${environment.storage_root_source}`),
  })
})

const form = reactive<SystemSettings>({
  registration_mode: 'admin_only',
  email: {
    mode: 'smtp',
    host: '',
    port: 587,
    username: '',
    password: '',
    password_set: false,
    from: '',
    from_name: 'kageos',
  },
})

const configuredProviderCount = computed(() => authProviders.value.filter((provider) => provider.configured).length)
const enabledProviderCount = computed(() => authProviders.value.filter((provider) => provider.enabled).length)

function applySettings(settings: SystemSettings) {
  form.registration_mode = settings.registration_mode
  form.email = {
    ...settings.email,
    password: '',
  }
}

function applyProviderConfigDraft(provider: AuthLoginProviderInfo) {
  const config: Record<string, string> = {}
  provider.fields.forEach((field) => {
    config[field.key] = field.secret ? '' : field.value || ''
  })
  providerConfigs[provider.code] = config
}

function replaceProvider(updated: AuthLoginProviderInfo) {
  const index = authProviders.value.findIndex((item) => item.code === updated.code)
  if (index >= 0) {
    authProviders.value[index] = updated
  } else {
    authProviders.value.push(updated)
  }
  applyProviderConfigDraft(updated)
}

async function loadSettings() {
  loading.value = true
  try {
    applySettings(await getSystemSettings())
  } catch (error: any) {
    ElMessage.error(error?.response?.data?.msg || error?.message || t('systemSettings.loadSettingsFailed'))
  } finally {
    loading.value = false
  }
}

async function loadAuthProviders() {
  providersLoading.value = true
  try {
    const [providersResult, announcementResult] = await Promise.allSettled([
      listAuthLoginProviders(),
      getLoginAnnouncementConfig(),
    ])
    if (providersResult.status === 'fulfilled') {
      authProviders.value = providersResult.value.providers || []
      authProviders.value.forEach(applyProviderConfigDraft)
    } else {
      const error: any = providersResult.reason
      ElMessage.error(error?.response?.data?.msg || error?.message || t('systemSettings.loadAuthProvidersFailed'))
    }
    if (announcementResult.status === 'fulfilled') {
      Object.assign(loginAnnouncement, announcementResult.value)
    } else {
      const error: any = announcementResult.reason
      ElMessage.error(error?.response?.data?.msg || error?.message || t('systemSettings.loginAnnouncementLoadFailed'))
    }
  } finally {
    providersLoading.value = false
  }
}

async function saveLoginAnnouncement() {
  if (loginAnnouncement.enabled && !loginAnnouncement.markdown.trim()) {
    ElMessage.warning(t('systemSettings.loginAnnouncementContentRequired'))
    return
  }
  announcementSaving.value = true
  try {
    Object.assign(loginAnnouncement, await updateLoginAnnouncementConfig({ ...loginAnnouncement }))
    ElMessage.success(t('systemSettings.loginAnnouncementSaved'))
  } catch (error: any) {
    ElMessage.error(error?.response?.data?.msg || error?.message || t('systemSettings.loginAnnouncementSaveFailed'))
  } finally {
    announcementSaving.value = false
  }
}

async function refreshActiveTab() {
  if (activeTab.value === 'operations') {
    await loadResources()
    return
  }
  if (activeTab.value === 'login') {
    await loadAuthProviders()
    return
  }
  if (activeTab.value === 'connectors') {
    connectorPanelKey.value += 1
    return
  }
  if (activeTab.value === 'openapi') {
    openapiPanelKey.value += 1
    return
  }
  if (activeTab.value === 'users') {
    usersPanelKey.value += 1
    return
  }
  if (activeTab.value === 'backups') {
    await loadArchives()
    return
  }
  if (activeTab.value === 'appearance' || activeTab.value === 'language') {
    return
  }
  await loadSettings()
}

function handleTabChange(tabName: string | number) {
  if (tabName === 'operations' && !resourceOverview.value) {
    loadResources()
  }
  if (tabName === 'login' && !authProviders.value.length) {
    loadAuthProviders()
  }
  if (tabName === 'backups' && !archiveBatches.value.length) {
    loadArchives()
  }
}

async function loadResources() {
  resourcesLoading.value = true
  try {
    resourceOverview.value = await getSystemResourceOverview(resourceHistoryHours.value)
    lastFullResourceRefreshAt = Date.now()
  } catch (error: any) {
    ElMessage.error(error?.response?.data?.msg || error?.message || t('systemSettings.resources.loadFailed'))
  } finally {
    resourcesLoading.value = false
  }
}

async function refreshResourcesSilently() {
  if (activeTab.value !== 'operations' || resourcesLoading.value) return
  try {
    const previous = resourceOverview.value
    const includeHistory = !previous || Date.now() - lastFullResourceRefreshAt >= 10 * 60_000
    const latest = await getSystemResourceOverview(resourceHistoryHours.value, includeHistory)
    if (includeHistory || !previous) {
      resourceOverview.value = latest
      lastFullResourceRefreshAt = Date.now()
    } else {
      resourceOverview.value = {
        ...latest,
        history: previous.history,
        history_hours: previous.history_hours,
        forecast: previous.forecast
      }
    }
  } catch {
    // Keep the last successful snapshot; the explicit refresh path reports errors.
  }
}

function formatBytes(value: number) {
  if (!Number.isFinite(value) || value < 0) return '-'
  const units = ['B', 'KB', 'MB', 'GB', 'TB', 'PB']
  let amount = value
  let unit = 0
  while (amount >= 1024 && unit < units.length - 1) {
    amount /= 1024
    unit += 1
  }
  return `${amount.toFixed(unit === 0 ? 0 : amount >= 100 ? 0 : 1)} ${units[unit]}`
}

function formatRate(value: number) {
  return `${formatBytes(value)}/s`
}

function roundedPercent(value: number) {
  return Math.min(100, Math.max(0, Number(value.toFixed(1))))
}

function progressStatus(value: number): 'success' | 'warning' | 'exception' | undefined {
  if (value >= 90) return 'exception'
  if (value >= 80) return 'warning'
  if (value < 60) return 'success'
  return undefined
}

function componentDiskPercent(bytes: number, poolKey: string) {
  const total = resourceOverview.value?.current.storage_pools.find((pool) => pool.key === poolKey)?.total_bytes || 0
  return total > 0 ? roundedPercent(bytes * 100 / total) : 0
}

function storagePoolName(key: string, fallback: string) {
  const translated = t(`systemSettings.resources.pools.${key}`)
  return translated.includes('systemSettings.resources.pools.') ? fallback : translated
}

function resourceComponentName(key: string, fallback: string) {
  const translated = t(`systemSettings.resources.components.${key}`)
  return translated.includes('systemSettings.resources.components.') ? fallback : translated
}

function formatUptime(seconds: number) {
  const days = Math.floor(seconds / 86400)
  const hours = Math.floor((seconds % 86400) / 3600)
  return t('systemSettings.resources.uptime', { days, hours })
}

function formatResourceTime(value: string) {
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString()
}

function historyPolyline(key: keyof Pick<SystemResourceHistoryPoint, 'disk_used_percent' | 'memory_used_percent' | 'cpu_used_percent'>) {
  const history = resourceOverview.value?.history || []
  if (history.length < 2) return ''
  return history.map((point, index) => {
    const x = index * 800 / (history.length - 1)
    const y = 180 - Math.min(100, Math.max(0, point[key])) * 1.8
    return `${x.toFixed(1)},${y.toFixed(1)}`
  }).join(' ')
}

function historyRatePolyline(key: keyof Pick<SystemResourceHistoryPoint, 'network_rx_bytes_per_second' | 'network_tx_bytes_per_second'>) {
  const history = resourceOverview.value?.history || []
  if (history.length < 2) return ''
  const max = Math.max(1, ...history.flatMap(point => [point.network_rx_bytes_per_second, point.network_tx_bytes_per_second]))
  return history.map((point, index) => {
    const x = index * 800 / (history.length - 1)
    const y = 120 - Math.min(1, Math.max(0, point[key] / max)) * 112
    return `${x.toFixed(1)},${y.toFixed(1)}`
  }).join(' ')
}

function collectionTaskName(key: string) {
  return t(`systemSettings.resources.collectionTasks.${key}`)
}

function collectionStatusLabel(status: string) {
  return t(`systemSettings.resources.collectionStatuses.${status}`)
}

function collectionStatusType(status: string): 'success' | 'warning' | 'danger' | 'info' {
  if (status === 'success') return 'success'
  if (status === 'running' || status === 'partial') return 'warning'
  if (status === 'failed') return 'danger'
  return 'info'
}

function collectionResult(error?: string) {
  if (!error) return t('systemSettings.resources.collectionNormal')
  const knownErrors: Record<string, string> = {
    'one or more platform metric sources are unavailable': 'platformSourceUnavailable',
    'application database metric source is unavailable': 'databaseSourceUnavailable',
    'runtime metrics collected but history persistence failed': 'runtimePersistenceFailed'
  }
  const key = knownErrors[error]
  return key ? t(`systemSettings.resources.collectionErrors.${key}`) : error
}

function formatDurationMillis(value: number) {
  if (value < 1000) return `${value} ms`
  return `${(value / 1000).toFixed(1)} s`
}

async function loadArchives() {
  archivesLoading.value = true
  try {
    const resp = await listLogArchiveBatches(archivePage.value, archivePageSize)
    archiveBatches.value = resp.list || []
    archiveTotal.value = resp.total || 0
    archiveRetentionDays.value = resp.retention_days || 90
    archiveCronExpr.value = resp.cron_expr || '20 3 * * *'
    archiveTimezone.value = resp.timezone || 'Asia/Shanghai'
  } catch (error: any) {
    ElMessage.error(error?.response?.data?.msg || error?.message || t('systemSettings.loadArchivesFailed'))
  } finally {
    archivesLoading.value = false
  }
}

function formatArchiveTime(value?: string) {
  if (!value) return '-'
  const date = new Date(value.replace(' ', 'T'))
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString()
}

function formatFileSize(size: number) {
  if (!size) return '-'
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`
  return `${(size / 1024 / 1024).toFixed(1)} MB`
}

function archiveStatusType(status: string) {
  if (status === 'completed') return 'success'
  if (status === 'failed') return 'danger'
  return 'warning'
}

function archiveStatusLabel(status: string) {
  return t(`systemSettings.archiveStatuses.${status}`)
}

function archiveResourceSummary(batch: LogArchiveBatch) {
  const paths = batch.summary_json?.top_resource_paths || []
  if (!paths.length) return '-'
  return paths.slice(0, 3).map((item) => `${item.resource_path || '/'} (${item.count})`).join('；')
}

function isSettingsTab(value: unknown): value is SettingsTab {
  return typeof value === 'string' && settingsSections.value.some((section) => section.key === value)
}

function selectSettingsSection(tabName: SettingsTab) {
  activeTab.value = tabName
  handleTabChange(tabName)
  router.replace({
    path: route.path,
    query: { ...route.query, tab: tabName }
  })
}

function openCurrentDocs() {
  openExternalURL(currentDocsURL.value)
}

function handleThemeChange(themeName: string) {
  const theme = themeStore.getAvailableThemes().find((item) => item.name === themeName)
  if (theme) {
    themeStore.setTheme(theme)
  }
}

function handleLocaleChange(locale: string) {
  localeStore.setAppLocale(locale as SupportedLocale)
}

async function saveSettings() {
  saving.value = true
  try {
    applySettings(await updateSystemSettings(JSON.parse(JSON.stringify(form))))
    ElMessage.success(t('systemSettings.settingsSaved'))
  } catch (error: any) {
    ElMessage.error(error?.response?.data?.msg || error?.message || t('systemSettings.saveSettingsFailed'))
  } finally {
    saving.value = false
  }
}

function providerStatusType(provider: AuthLoginProviderInfo) {
  if (provider.status === 'enabled') {
    return 'success'
  }
  if (provider.status === 'disabled') {
    return 'warning'
  }
  return 'info'
}

function providerStatusLabel(provider: AuthLoginProviderInfo) {
  if (provider.status === 'enabled') {
    return t('systemSettings.statusEnabled')
  }
  if (provider.status === 'disabled') {
    return t('systemSettings.statusConfiguredDisabled')
  }
  return t('systemSettings.statusUnconfigured')
}

function providerActionLabel(action: string) {
  if (action === 'qrcode') {
    return t('systemSettings.actionQrcode')
  }
  if (action === 'redirect') {
    return t('systemSettings.actionRedirect')
  }
  return action || t('systemSettings.actionAuthorize')
}

function fieldInputType(field: AuthLoginProviderField) {
  if (field.secret) {
    return 'password'
  }
  if (field.type === 'url') {
    return 'url'
  }
  return 'text'
}

function fieldPlaceholder(field: AuthLoginProviderField) {
  if (field.secret && field.value_set) {
    return t('systemSettings.secretKeepPlaceholder')
  }
  if (field.placeholder) {
    return field.placeholder
  }
  return field.secret ? t('systemSettings.secretPlaceholder') : t('systemSettings.configValuePlaceholder')
}

function isCallbackField(field: AuthLoginProviderField) {
  return field.key === 'redirect_url' || field.key === 'callback_url'
}

function providerFieldValue(provider: AuthLoginProviderInfo, field: AuthLoginProviderField) {
  return providerConfigs[provider.code]?.[field.key] || ''
}

function providerBooleanFieldValue(provider: AuthLoginProviderInfo, field: AuthLoginProviderField) {
  return providerFieldValue(provider, field) === 'true'
}

function setProviderFieldValue(provider: AuthLoginProviderInfo, field: AuthLoginProviderField, value: string | number | boolean) {
  let config = providerConfigs[provider.code]
  if (!config) {
    config = {}
    providerConfigs[provider.code] = config
  }
  if (field.type === 'boolean') {
    config[field.key] = value ? 'true' : 'false'
    return
  }
  config[field.key] = String(value ?? '')
}

function callbackURL(provider: AuthLoginProviderInfo) {
  const path = provider.callback_path || ''
  if (!path) {
    return ''
  }
  if (/^https?:\/\//i.test(path)) {
    return path
  }
  if (typeof window === 'undefined') {
    return path
  }
  return `${window.location.origin}${path.startsWith('/') ? path : `/${path}`}`
}

function fillCallbackURL(provider: AuthLoginProviderInfo, fieldKey: string) {
  const config = providerConfigs[provider.code]
  if (!config) {
    return
  }
  config[fieldKey] = callbackURL(provider)
}

async function copyCallbackURL(provider: AuthLoginProviderInfo) {
  try {
    await navigator.clipboard.writeText(callbackURL(provider))
    ElMessage.success(t('connectorProvider.callbackCopied'))
  } catch {
    ElMessage.error(t('connectorProvider.copyFailed'))
  }
}

function buildProviderConfigPayload(provider: AuthLoginProviderInfo) {
  const draft = providerConfigs[provider.code] || {}
  const payload: Record<string, string> = {}
  provider.fields.forEach((field) => {
    const value = (draft[field.key] || '').trim()
    if (field.secret && !value) {
      return
    }
    payload[field.key] = value
  })
  return payload
}

function validateProviderConfig(provider: AuthLoginProviderInfo) {
  const draft = providerConfigs[provider.code] || {}
  const missing = provider.fields.find((field) => {
    if (!field.required) {
      return false
    }
    if (field.secret && field.value_set) {
      return false
    }
    return !(draft[field.key] || '').trim()
  })
  if (missing) {
    ElMessage.warning(t('systemSettings.providerFieldMissing', { provider: provider.name, field: missing.label }))
    return false
  }
  return true
}

async function saveProviderConfig(provider: AuthLoginProviderInfo) {
  if (!validateProviderConfig(provider)) {
    return
  }
  providerSaving[provider.code] = true
  try {
    const updated = await updateAuthLoginProviderConfig(provider.code, buildProviderConfigPayload(provider))
    replaceProvider(updated)
    ElMessage.success(t('systemSettings.providerSaved'))
  } catch (error: any) {
    ElMessage.error(error?.response?.data?.msg || error?.message || t('systemSettings.providerSaveFailed'))
  } finally {
    providerSaving[provider.code] = false
  }
}

async function handleProviderEnabledChange(provider: AuthLoginProviderInfo, enabled: boolean) {
  if (enabled && !provider.configured) {
    ElMessage.warning(t('systemSettings.saveBeforeEnable'))
    return
  }
  providerSwitching[provider.code] = true
  try {
    const updated = await updateAuthLoginProviderEnabled(provider.code, enabled)
    replaceProvider(updated)
    ElMessage.success(enabled ? t('systemSettings.providerEnabled') : t('systemSettings.providerDisabled'))
  } catch (error: any) {
    ElMessage.error(error?.response?.data?.msg || error?.message || t('systemSettings.providerStatusFailed'))
  } finally {
    providerSwitching[provider.code] = false
  }
}

function handleProviderSwitchChange(provider: AuthLoginProviderInfo, value: string | number | boolean) {
  handleProviderEnabledChange(provider, Boolean(value))
}

async function sendTestEmail() {
  if (!testEmail.value.trim()) {
    ElMessage.warning(t('systemSettings.recipientRequired'))
    return
  }
  testing.value = true
  try {
    await testSystemEmail(testEmail.value.trim())
    ElMessage.success(t('systemSettings.testEmailSent'))
  } catch (error: any) {
    ElMessage.error(error?.response?.data?.msg || error?.message || t('systemSettings.testEmailFailed'))
  } finally {
    testing.value = false
  }
}

onMounted(() => {
  const tab = Array.isArray(route.query.tab) ? route.query.tab[0] : route.query.tab
  if (isSettingsTab(tab)) {
    activeTab.value = tab
  } else if (!isSettingsTab(activeTab.value)) {
    activeTab.value = settingsSections.value[0]?.key || defaultSettingsTab
  }
  if (activeTab.value === 'email') {
    loadSettings()
  } else {
    handleTabChange(activeTab.value)
  }
  resourceRefreshTimer = setInterval(refreshResourcesSilently, 30_000)
})

onBeforeUnmount(() => {
  if (resourceRefreshTimer) clearInterval(resourceRefreshTimer)
})
</script>

<style scoped>
.system-settings-page {
  min-height: 100vh;
  padding: 32px clamp(20px, 2.5vw, 48px);
  background: var(--bg-primary);
}

.settings-card {
  width: 100%;
  border-radius: var(--border-radius-xl);
  border: 1px solid var(--border-light);
  box-shadow: var(--app-shell-panel-shadow-soft);
  background: var(--bg-secondary);
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 24px 32px;
  border-bottom: 1px solid var(--border-light);
}

.card-header h2 {
  margin: 0 0 6px;
  font-size: 20px;
  font-weight: 600;
  color: var(--text-primary);
}

.card-header p {
  margin: 0;
  color: var(--text-secondary);
  font-size: 14px;
}

.header-actions,
.test-row {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.settings-layout {
  display: grid;
  grid-template-columns: 280px minmax(0, 1fr);
  gap: 0;
  align-items: stretch;
}

.settings-sidebar {
  border-right: 1px solid var(--border-light);
  background: var(--bg-secondary);
  padding: 16px;
  display: flex;
  flex-direction: column;
  gap: 6px;
  border-radius: var(--border-radius-xl) 0 0 var(--border-radius-xl);
}

.settings-nav-item {
  width: 100%;
  min-height: 64px;
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 12px 14px;
  border: 1px solid transparent;
  border-radius: var(--border-radius-base);
  background: var(--bg-tertiary);
  color: var(--text-primary);
  cursor: pointer;
  text-align: left;
  transition: all 0.25s cubic-bezier(0.25, 0.8, 0.25, 1);
}

.settings-nav-item:hover {
  background: var(--el-fill-color);
}

.settings-nav-item.is-active {
  background: var(--el-fill-color-blank);
  border-color: var(--color-primary);
  box-shadow: 0 0 0 2px rgba(var(--color-primary-rgb), 0.1);
}

.settings-nav-title {
  display: block;
  font-size: 14px;
  font-weight: 600;
  line-height: 1.3;
}

.settings-nav-desc {
  display: block;
  margin-top: 4px;
  color: var(--text-secondary);
  font-size: 12px;
  line-height: 1.35;
}

.settings-content {
  min-width: 0;
  background: transparent;
  border-radius: 0 var(--border-radius-xl) var(--border-radius-xl) 0;
  padding: 32px 40px;
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 16px;
  padding-bottom: 20px;
  margin-bottom: 24px;
  border-bottom: 1px solid var(--border-light);
}

.section-header h3 {
  margin: 0 0 8px;
  font-size: 18px;
  font-weight: 600;
  color: var(--text-primary);
}

.section-header p {
  margin: 0;
  color: var(--text-secondary);
  font-size: 14px;
  line-height: 1.5;
}

.section-pane {
  min-width: 0;
}

.settings-form {
  max-width: 760px;
}

.archive-summary-row {
  display: flex;
  flex-wrap: wrap;
  gap: 12px 24px;
  margin: 16px 0;
  color: var(--text-secondary);
  font-size: 13px;
}

.archive-table {
  margin-bottom: 16px;
}

.archive-evidence {
  display: grid;
  gap: 4px;
  font-size: 11px;
  overflow-wrap: anywhere;
}

.test-row {
  width: 100%;
}

.login-provider-section {
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-height: 160px;
}

.login-announcement-panel {
  padding: 18px;
  border: 1px solid var(--el-border-color-light);
  border-radius: 8px;
  background: var(--el-bg-color);
}

.login-announcement-heading {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 20px;
  margin-bottom: 18px;
}

.login-announcement-heading h4 {
  margin: 0;
  font-size: 16px;
  color: var(--text-primary);
}

.login-announcement-heading p {
  margin: 8px 0 0;
  color: var(--text-secondary);
  line-height: 1.5;
}

.login-announcement-actions {
  display: flex;
  justify-content: flex-end;
}

.provider-summary {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
}

.summary-item {
  min-width: 0;
  padding: 14px 16px;
  border: 1px solid var(--el-border-color-light);
  border-radius: 8px;
  background: var(--el-fill-color-lighter);
}

.summary-value {
  display: block;
  font-size: 22px;
  font-weight: 700;
  color: var(--el-text-color-primary);
  line-height: 1.2;
}

.summary-label {
  display: block;
  margin-top: 4px;
  font-size: 13px;
  color: var(--el-text-color-secondary);
}

.provider-panel {
  padding: 18px;
  border: 1px solid var(--el-border-color-light);
  border-radius: 8px;
  background: var(--el-bg-color);
}

.provider-panel-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 20px;
  margin-bottom: 18px;
}

.provider-heading {
  min-width: 0;
  flex: 1;
}

.provider-title-row {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.provider-name {
  font-size: 16px;
  font-weight: 600;
  color: var(--text-primary);
}

.provider-heading p {
  margin: 8px 0 0;
  color: var(--text-secondary);
  line-height: 1.5;
}

.provider-meta {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 10px;
  font-size: 13px;
  color: var(--text-secondary);
}

.callback-path {
  min-width: 0;
  word-break: break-all;
}

.callback-path code {
  padding: 2px 6px;
  border-radius: 4px;
  background: var(--el-fill-color-light);
  color: var(--text-primary);
}

.provider-enable {
  display: flex;
  align-items: center;
  gap: 10px;
  color: var(--text-secondary);
  white-space: nowrap;
}

.provider-form {
  max-width: 860px;
}

.provider-field {
  width: 100%;
}

.field-help {
  margin-top: 6px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  line-height: 1.5;
}

.provider-form-actions {
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
}

.preference-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 12px;
  max-width: 720px;
}

.preference-card {
  min-height: 82px;
  padding: 16px 20px;
  border: 1px solid transparent;
  border-radius: var(--border-radius-lg);
  background: var(--bg-tertiary);
  color: var(--text-regular);
  text-align: left;
  cursor: pointer;
  transition: all 0.25s cubic-bezier(0.25, 0.8, 0.25, 1);
}

.preference-card:hover {
  background: var(--el-fill-color);
}

.preference-card.is-active {
  background: var(--el-fill-color-blank);
  border-color: var(--color-primary);
  box-shadow: 0 0 0 2px rgba(var(--color-primary-rgb), 0.15);
  color: var(--color-primary);
}

.preference-card-title {
  display: block;
  font-size: 15px;
  font-weight: 700;
}

.preference-card-desc {
  display: block;
  margin-top: 8px;
  color: var(--el-text-color-secondary);
  font-size: 13px;
  line-height: 1.45;
}

.operations-pane { display: grid; gap: 20px; }

.resource-summary-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 16px;
}

.resource-summary-card,
.resource-panel {
  border: 1px solid var(--border-light);
  border-radius: var(--border-radius-lg);
  background: var(--bg-primary);
}

.resource-summary-card {
  display: flex;
  flex-direction: column;
  gap: 10px;
  min-width: 0;
  padding: 18px;
}

.resource-summary-card strong {
  overflow: hidden;
  color: var(--text-primary);
  font-size: 18px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.resource-summary-card small,
.resource-panel-heading p,
.resource-collected-at { color: var(--text-secondary); }

.resource-summary-label {
  color: var(--text-secondary);
  font-size: 13px;
  font-weight: 600;
}

.resource-panel { padding: 20px; }

.storage-pool-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
  gap: 14px;
}

.storage-pool-card {
  display: grid;
  gap: 10px;
  padding: 16px;
  border: 1px solid var(--border-light);
  border-radius: var(--border-radius-lg);
  background: var(--bg-secondary);
}

.storage-pool-title {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.storage-pool-amount { color: var(--text-primary); font-size: 16px; }
.storage-pool-card small { color: var(--text-secondary); }

.platform-metric-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 12px;
  margin-bottom: 16px;
}

.platform-metric-card {
  display: grid;
  gap: 7px;
  padding: 14px;
  border: 1px solid var(--border-light);
  border-radius: var(--border-radius-lg);
  background: var(--bg-secondary);
}

.platform-metric-card span,
.platform-metric-card small { color: var(--text-secondary); font-size: 12px; }
.platform-metric-card strong { color: var(--text-primary); font-size: 24px; }
.database-size-table { margin-top: 8px; }
.collection-error { color: var(--el-color-danger); }

.resource-panel-heading {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 18px;
}

.resource-panel-heading h4,
.resource-panel-heading p { margin: 0; }
.resource-panel-heading p { margin-top: 5px; font-size: 13px; }
.resource-chart-wrap { overflow: hidden; }
.resource-chart { display: block; width: 100%; height: 180px; }
.chart-grid-line { stroke: var(--border-light); stroke-width: 1; }

.chart-line {
  fill: none;
  stroke-linecap: round;
  stroke-linejoin: round;
  stroke-width: 3;
  vector-effect: non-scaling-stroke;
}

.disk-line { stroke: var(--el-color-primary); }
.memory-line { stroke: var(--el-color-warning); }
.cpu-line { stroke: var(--el-color-success); }
.network-rx-line { stroke: var(--el-color-primary); }
.network-tx-line { stroke: var(--el-color-danger); }
.rate-chart { height: 120px; margin-top: 22px; border-top: 1px solid var(--border-light); }

.chart-legend {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
  margin-top: 12px;
  color: var(--text-secondary);
  font-size: 12px;
}

.chart-legend span:last-child { margin-left: auto; }
.legend-dot { display: inline-block; width: 8px; height: 8px; margin-right: 6px; border-radius: 50%; }
.disk-dot { background: var(--el-color-primary); }
.memory-dot { background: var(--el-color-warning); }
.cpu-dot { background: var(--el-color-success); }
.network-rx-dot { background: var(--el-color-primary); }
.network-tx-dot { background: var(--el-color-danger); }
.resource-collected-at { margin: -6px 0 0; text-align: right; font-size: 12px; }

@media (max-width: 768px) {
  .system-settings-page {
    padding: 12px;
  }

  .settings-layout {
    grid-template-columns: 1fr;
  }

  .settings-sidebar {
    position: static;
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .card-header,
  .provider-panel-header {
    flex-direction: column;
    align-items: stretch;
  }

  .provider-summary {
    grid-template-columns: 1fr;
  }

  .resource-summary-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .settings-form,
  .provider-form {
    max-width: none;
  }

  .test-row {
    flex-direction: column;
    align-items: stretch;
  }
}

@media (max-width: 520px) {
  .settings-sidebar,
  .preference-grid,
  .resource-summary-grid {
    grid-template-columns: 1fr;
  }

  .resource-panel-heading { flex-direction: column; }
}
</style>
