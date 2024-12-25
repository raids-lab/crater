import { themes as prismThemes } from "prism-react-renderer";
import type { Config } from "@docusaurus/types";
import type * as Preset from "@docusaurus/preset-classic";
import tailwindPlugin from "./plugins/tailwind-config.cjs";

// This runs in Node.js - Don't use client-side code here (browser APIs, JSX...)

const config: Config = {
  title: "Crater",
  tagline: "A GPU Resource Scheduling System based on Kubernetes",
  favicon: "img/logo.svg",

  // Set the production url of your site here
  url: "https://gpu.act.buaa.edu.cn",
  // Set the /<baseUrl>/ pathname under which your site is served
  // For GitHub pages deployment, it is often '/<projectName>/'
  baseUrl: "/website/",

  // GitHub pages deployment config.
  // If you aren't using GitHub pages, you don't need these.
  organizationName: "radis-lab", // Usually your GitHub org/user name.
  projectName: "crater-document", // Usually your repo name.

  onBrokenLinks: "throw",
  onBrokenMarkdownLinks: "warn",

  markdown: {
    mermaid: true,
  },

  plugins: [tailwindPlugin],

  // Even if you don't use internationalization, you can use this field to set
  // useful metadata like html lang. For example, if your site is Chinese, you
  // may want to replace "en" with "zh-Hans".
  i18n: {
    defaultLocale: "zh-CN",
    locales: ["zh-CN", "en-US"],
  },

  presets: [
    [
      "classic",
      {
        docs: {
          sidebarPath: "./sidebars.ts",
          // Please change this to your repo.
          // Remove this to remove the "edit this page" links.
          // editUrl:
          //   "https://github.com/facebook/docusaurus/tree/main/packages/create-docusaurus/templates/shared/",
        },
        blog: {
          showReadingTime: true,
          feedOptions: {
            type: ["rss", "atom"],
            xslt: true,
          },
          // Please change this to your repo.
          // Remove this to remove the "edit this page" links.
          // editUrl:
          //   "https://github.com/facebook/docusaurus/tree/main/packages/create-docusaurus/templates/shared/",
          // Useful options to enforce blogging best practices
          onInlineTags: "warn",
          onInlineAuthors: "warn",
          onUntruncatedBlogPosts: "warn",
        },
        theme: {
          customCss: "./src/css/custom.css",
        },
      } satisfies Preset.Options,
    ],
  ],

  themeConfig: {
    // Replace with your project's social card
    image: "img/docusaurus-social-card.jpg",
    navbar: {
      title: "Crater",
      logo: {
        alt: "Crater Logo",
        src: "img/logo.svg",
      },
      items: [
        {
          type: "docSidebar",
          sidebarId: "tutorialSidebar",
          position: "left",
          label: "文档",
        },
        { to: "/blog", label: "资讯", position: "left" },
        {
          href: "https://gpu.act.buaa.edu.cn/portal",
          label: "立即体验",
          position: "right",
        },
        {
          href: "https://gitlab.act.buaa.edu.cn/groups/raids/resource-scheduling/crater",
          label: "Gitlab",
          position: "right",
        },
      ],
    },
    footer: {
      style: "dark",
      links: [
        {
          title: "文档",
          items: [
            {
              label: "文档",
              to: "/docs/intro",
            },
          ],
        },
        {
          title: "社区",
          items: [
            {
              label: "档案",
              href: "https://docs.qq.com/desktop/mydoc/folder/XnmrOEBITHOs",
            },
            {
              label: "月报",
              href: "https://docs.qq.com/aio/DUHplWVB6cVJqSlFZ",
            },
            {
              label: "资讯",
              to: "/blog",
            },
          ],
        },
        {
          title: "更多",
          items: [
            {
              label: "Gitlab",
              href: "https://gitlab.act.buaa.edu.cn/groups/raids/resource-scheduling/crater",
            },
          ],
        },
      ],
      copyright: `Copyright © ${new Date().getFullYear()} ACT RAIDS Lab. Built with Docusaurus.`,
    },
    prism: {
      theme: prismThemes.github,
      darkTheme: prismThemes.dracula,
    },
  } satisfies Preset.ThemeConfig,

  // ... Your other configurations.
  themes: [
    [
      require.resolve("@easyops-cn/docusaurus-search-local"),
      /** @type {import("@easyops-cn/docusaurus-search-local").PluginOptions} */
      {
        hashed: true,
        language: ["en", "zh"],
      },
    ],
    "@docusaurus/theme-mermaid",
  ],
};

export default config;
