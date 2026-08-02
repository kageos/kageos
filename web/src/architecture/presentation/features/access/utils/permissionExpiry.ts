export function isPermissionExpiryValid(
  permanent: boolean,
  expiresAt: Date | null | undefined,
  now = new Date(),
): boolean {
  if (permanent) return true
  return expiresAt instanceof Date
    && Number.isFinite(expiresAt.getTime())
    && expiresAt.getTime() > now.getTime()
}

export function disablePastPermissionDate(date: Date, now = new Date()): boolean {
  const startOfToday = new Date(now.getFullYear(), now.getMonth(), now.getDate())
  return date.getTime() < startOfToday.getTime()
}
