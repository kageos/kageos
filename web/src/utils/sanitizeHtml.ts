const BLOCKED_TAGS = new Set(['script', 'iframe', 'object', 'embed', 'template', 'style'])
const URL_ATTRS = new Set(['href', 'src', 'poster', 'xlink:href'])

function isSafeUrl(url: string): boolean {
  const value = url.trim()
  if (!value) return true
  return /^(https?:|mailto:|tel:|\/|#|blob:|data:image\/(?:png|jpeg|gif|webp|svg\+xml);)/i.test(value)
}

function sanitizeSrcset(srcset: string): string {
  return srcset
    .split(',')
    .map((candidate) => candidate.trim())
    .filter(Boolean)
    .filter((candidate) => {
      const url = candidate.split(/\s+/, 1)[0] || ''
      return isSafeUrl(url)
    })
    .join(', ')
}

export function escapeHtml(content: string): string {
  return content
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
}

export function sanitizeHtml(html: string): string {
  if (!html) return ''

  if (typeof DOMParser === 'undefined') {
    return escapeHtml(html)
  }

  const parser = new DOMParser()
  const doc = parser.parseFromString(html, 'text/html')
  const elements = Array.from(doc.body.querySelectorAll('*'))

  for (const element of elements) {
    const tagName = element.tagName.toLowerCase()
    if (BLOCKED_TAGS.has(tagName)) {
      element.remove()
      continue
    }

    for (const attr of Array.from(element.attributes)) {
      const attrName = attr.name.toLowerCase()
      const attrValue = attr.value

      if (attrName.startsWith('on') || attrName === 'srcdoc') {
        element.removeAttribute(attr.name)
        continue
      }

      if (attrName === 'srcset') {
        const safeSrcset = sanitizeSrcset(attrValue)
        if (safeSrcset) {
          element.setAttribute(attr.name, safeSrcset)
        } else {
          element.removeAttribute(attr.name)
        }
        continue
      }

      if (URL_ATTRS.has(attrName) && !isSafeUrl(attrValue)) {
        element.removeAttribute(attr.name)
      }
    }
  }

  return doc.body.innerHTML
}
