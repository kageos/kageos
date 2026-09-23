type Translate = (key: string, values?: Record<string, string | number>) => string

export interface WorkspaceUpdateLog {
  status?: string
  details_json?: {
    outcome?: string
    write_only?: boolean
    force_diff?: boolean
    warnings?: string[]
    error?: string
    change_description?: string
    git_commit_hash?: string
    build_trace_id?: string
    changes?: {
      mode: string
      added: WorkspaceFunctionChange[]
      updated: WorkspaceFunctionChange[]
      deleted: WorkspaceFunctionChange[]
      synced: WorkspaceFunctionChange[]
    }
  }
  old_values_json?: { version?: string }
  new_values_json?: { version?: string }
}

export interface WorkspaceFunctionChange {
  name: string
  path: string
  type: string
}

export function workspaceUpdateHasWarnings(log: WorkspaceUpdateLog): boolean {
  return !!log.details_json?.warnings?.length
}

export function workspaceUpdateTitle(log: WorkspaceUpdateLog, t: Translate): string {
  const details = log.details_json
  if (details?.outcome === 'finalization_failed' && !details.write_only) return t('workspaceUpdateLog.finalizationFailed')
  if (log.status === 'failed') return t('operateLog.workspaceUpdateFailed')
  if (details?.write_only) return t('workspaceUpdateLog.writeOnly')
  if (workspaceUpdateHasWarnings(log)) return t('workspaceUpdateLog.publishedWarning')
  return t('operateLog.workspaceUpdated')
}

export function workspaceUpdateSummary(log: WorkspaceUpdateLog, t: Translate): string {
  const details = log.details_json
  const changes = details?.changes
  const oldVersion = log.old_values_json?.version
  const newVersion = log.new_values_json?.version
  const version = oldVersion && newVersion ? `${oldVersion} → ${newVersion}` : ''
  let summary: string
  if (details?.write_only) {
    summary = t(log.status === 'failed' ? 'workspaceUpdateLog.writeOnlyFailedHint' : 'workspaceUpdateLog.writeOnlyHint')
  } else if (!changes) {
    summary = t('workspaceUpdateLog.unavailable')
  } else if (details?.force_diff || changes.mode === 'resync') {
    summary = t('workspaceUpdateLog.resyncSummary', { count: changes.synced?.length ?? 0 })
  } else {
    const added = changes.added?.length ?? 0
    const updated = changes.updated?.length ?? 0
    const deleted = changes.deleted?.length ?? 0
    summary = added + updated + deleted === 0
      ? t('workspaceUpdateLog.empty')
      : t('workspaceUpdateLog.counts', { added, updated, deleted })
  }
  if (log.status === 'failed' || workspaceUpdateHasWarnings(log)) {
    summary = `${t('workspaceUpdateLog.incomplete')} · ${summary}`
  }
  return [version, summary].filter(Boolean).join(' · ')
}
