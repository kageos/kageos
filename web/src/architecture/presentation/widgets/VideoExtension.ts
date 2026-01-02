import { Node, mergeAttributes } from '@tiptap/core'

export interface VideoOptions {
  inline: boolean
  allowBase64: boolean
  HTMLAttributes: Record<string, any>
}

declare module '@tiptap/core' {
  interface Commands<ReturnType> {
    video: {
      /**
       * Add a video
       */
      setVideo: (options: { src: string; alt?: string; title?: string }) => ReturnType
    }
  }
}

export const Video = Node.create<VideoOptions>({
  name: 'video',

  addOptions() {
    return {
      inline: false,
      allowBase64: false,
      HTMLAttributes: {},
    }
  },

  inline() {
    return this.options.inline
  },

  group() {
    return this.options.inline ? 'inline' : 'block'
  },

  draggable: true,

  addAttributes() {
    return {
      src: {
        default: null,
      },
      alt: {
        default: null,
      },
      title: {
        default: null,
      },
      controls: {
        default: true,
      },
      autoplay: {
        default: false,
      },
      loop: {
        default: false,
      },
      muted: {
        default: false,
      },
      playsinline: {
        default: false,
      },
    }
  },

  parseHTML() {
    return [
      {
        tag: 'video',
        getAttrs: (node) => {
          if (typeof node === 'string') return false
          const element = node as HTMLElement
          
          // 从 video 标签的 src 属性获取
          let src = element.getAttribute('src')
          
          // 如果没有 src 属性，尝试从 source 子元素获取
          if (!src) {
            const source = element.querySelector('source')
            if (source) {
              src = source.getAttribute('src')
            }
          }
          
          return {
            src,
            alt: element.getAttribute('alt'),
            title: element.getAttribute('title'),
            controls: element.hasAttribute('controls'),
            autoplay: element.hasAttribute('autoplay'),
            loop: element.hasAttribute('loop'),
            muted: element.hasAttribute('muted'),
            playsinline: element.hasAttribute('playsinline'),
          }
        },
      },
    ]
  },

  renderHTML({ HTMLAttributes }) {
    const { src, ...restAttrs } = HTMLAttributes
    const videoAttrs = mergeAttributes(this.options.HTMLAttributes, restAttrs, {
      style: 'max-width: 100%; height: auto; border-radius: 4px; margin: 8px 0; display: block; background-color: #000;',
    })
    
    if (src) {
      return [
        'video',
        videoAttrs,
        ['source', { src }],
      ]
    }
    
    return ['video', videoAttrs]
  },

  addCommands() {
    return {
      setVideo: (options) => ({ commands }) => {
        return commands.insertContent({
          type: this.name,
          attrs: options,
        })
      },
    }
  },
})

