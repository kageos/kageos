const svgDataURL = (svg: string) => `data:image/svg+xml,${encodeURIComponent(svg)}`

const googleLogo = svgDataURL(`
  <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 48 48">
    <path fill="#FFC107" d="M43.6 20.1H42V20H24v8h11.3c-1.6 4.7-6.1 8-11.3 8-6.6 0-12-5.4-12-12s5.4-12 12-12c3.1 0 5.9 1.2 8 3.1l5.7-5.7A20 20 0 0 0 4 24c0 11 9 20 20 20s20-9 20-20c0-1.3-.1-2.6-.4-3.9Z"/>
    <path fill="#FF3D00" d="m6.3 14.7 6.6 4.8A12 12 0 0 1 24 12c3.1 0 5.9 1.2 8 3.1l5.7-5.7A20 20 0 0 0 6.3 14.7Z"/>
    <path fill="#4CAF50" d="M24 44c5.1 0 9.8-1.9 13.4-5.2l-6.2-5.2A11.9 11.9 0 0 1 12.7 28l-6.5 5A20 20 0 0 0 24 44Z"/>
    <path fill="#1976D2" d="M43.6 20.1H42V20H24v8h11.3a12 12 0 0 1-4.1 5.6l6.2 5.2C41.2 35.3 44 30.3 44 24c0-1.3-.1-2.6-.4-3.9Z"/>
  </svg>
`)

const githubLogo = svgDataURL(`
  <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24">
    <circle cx="12" cy="12" r="12" fill="#24292f"/>
    <path fill="#fff" d="M12 5.3a6.8 6.8 0 0 0-2.1 13.2c.3.1.5-.1.5-.3v-1.3c-1.8.4-2.2-.8-2.2-.8-.3-.7-.7-.9-.7-.9-.6-.4 0-.4 0-.4.7 0 1 .7 1 .7.6 1 1.6.7 2 .5.1-.4.2-.7.4-.9-1.5-.2-3-.7-3-3.4 0-.8.3-1.4.7-1.9-.1-.2-.3-.9.1-1.9 0 0 .6-.2 1.9.7a6.5 6.5 0 0 1 3.5 0c1.3-.9 1.9-.7 1.9-.7.4 1 .2 1.7.1 1.9.5.5.7 1.1.7 1.9 0 2.7-1.6 3.2-3 3.4.2.2.4.6.4 1.2v1.9c0 .2.2.4.5.3A6.8 6.8 0 0 0 12 5.3Z"/>
  </svg>
`)

const wechatLogo = svgDataURL(`
  <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 48 48">
    <circle cx="24" cy="24" r="23" fill="#07C160"/>
    <path fill="#fff" d="M22 11c-7.2 0-13 4.8-13 10.8 0 3.4 1.9 6.5 5 8.5l-1.2 4 4.7-2.3c1.4.4 2.9.6 4.5.6h.8a9.8 9.8 0 0 1-.5-3c0-6 5.4-10.8 12.2-10.8h.2C33.1 14.3 28.1 11 22 11Zm-4.5 6.5a1.8 1.8 0 1 1 0 3.6 1.8 1.8 0 0 1 0-3.6Zm9 0a1.8 1.8 0 1 1 0 3.6 1.8 1.8 0 0 1 0-3.6Z"/>
    <path fill="#fff" d="M39.9 29.7c0-5-4.8-9-10.7-9s-10.7 4-10.7 9 4.8 9 10.7 9c1.2 0 2.4-.2 3.5-.5l4 2-1-3.4c2.6-1.7 4.2-4.2 4.2-7.1Zm-14.4-1.2a1.5 1.5 0 1 1 0-3 1.5 1.5 0 0 1 0 3Zm7.4 0a1.5 1.5 0 1 1 0-3 1.5 1.5 0 0 1 0 3Z"/>
  </svg>
`)

const genericLoginLogo = svgDataURL(`
  <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 48 48">
    <rect x="2" y="2" width="44" height="44" rx="12" fill="#E8F1FF"/>
    <path fill="none" stroke="#1677FF" stroke-width="4" stroke-linecap="round" stroke-linejoin="round" d="M27 13a8 8 0 1 0 0 16 8 8 0 0 0 0-16Zm-6 14L10 38m5-5 4 4"/>
  </svg>
`)

export const authProviderLogos = {
  google_oauth: googleLogo,
  github_oauth: githubLogo,
  kageos_auth: '/brand/kageos-avatar-128.png',
  wechat_open_oauth: wechatLogo,
  wechat_official: wechatLogo,
} as const

export function authProviderLogo(provider: string): string {
  const code = provider.trim().toLowerCase()
  if (code in authProviderLogos) {
    return authProviderLogos[code as keyof typeof authProviderLogos]
  }
  if (code.includes('google')) return googleLogo
  if (code.includes('github')) return githubLogo
  if (code.includes('wechat') || code.includes('weixin')) return wechatLogo
  if (code.includes('kageos')) return authProviderLogos.kageos_auth
  return genericLoginLogo
}
