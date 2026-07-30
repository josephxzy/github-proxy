import { defineConfig } from 'astro/config'
import starlight from '@astrojs/starlight'

export default defineConfig({
  site: 'https://josephxzy.github.io/github-proxy',
  base: process.env.BASE_URL || '/',
  integrations: [
    starlight({
      title: 'Github Proxy',
      description: '轻量级 GitHub 资源加速反向代理',
      defaultLocale: 'zh-CN',
      logo: {
        alt: 'Github Proxy',
        src: './src/assets/logo.svg',
      },
      social: [
        {
          icon: 'github',
          label: 'GitHub',
          href: 'https://github.com/josephxzy/github-proxy',
        },
      ],
      sidebar: [
        {
          label: '简介',
          link: '/',
        },
        {
          label: '快速开始',
          collapsed: true,
          items: [
            { label: '安装部署', link: '/getting-started/install/' },
          ],
        },
        {
          label: '配置',
          collapsed: true,
          items: [
            { label: '配置参考', link: '/configuration/reference/' },
            { label: '仓库黑白名单', link: '/configuration/repo-list/' },
            { label: 'Token 白名单', link: '/configuration/token-whitelist/' },
          ],
        },
        {
          label: '使用指南',
          collapsed: true,
          items: [
            { label: 'URL 格式', link: '/guides/url-format/' },
            { label: 'Git 加速', link: '/guides/git/' },
            { label: '私有仓库', link: '/guides/private-repo/' },
            { label: '仓库搜索', link: '/guides/search/' },
          ],
        },
        {
          label: '设计文档',
          collapsed: true,
          items: [
            { label: 'Token 透传与认证', link: '/design/token/' },
            { label: '下载与断点续传', link: '/design/download/' },
            { label: '限速与稳定性', link: '/design/rate-limit/' },
            { label: 'IP 限流设计', link: '/design/ip-limit/' },
          ],
        },
      ],
      customCss: ['./src/styles/custom.css'],
      head: [
        {
          tag: 'link',
          attrs: { rel: 'icon', type: 'image/svg+xml', href: '/favicon.svg' },
        },
      ],
    }),
  ],
})
