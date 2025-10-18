import { DocsThemeConfig } from 'nextra-theme-docs'

const config: DocsThemeConfig = {
  logo: <span>🥒 Gherkio Docs</span>,
  project: {
    link: 'https://github.com/muhfaris/gherkio'
  },
  docsRepositoryBase: 'https://github.com/muhfaris/gherkio/tree/main/docs',
  footer: {
    text: '© ' + new Date().getFullYear() + ' Gherkio'
  }
}

export default config
