import { defineConfig } from "vitepress";

export default defineConfig({
  title: "agentrunner",
  description:
    "Language-native libraries for programmatically invoking AI coding agents",
  base: "/agentrunner/",

  themeConfig: {
    nav: [
      { text: "Guide", link: "/guide/getting-started" },
      { text: "Libraries", link: "/libraries/go" },
      { text: "Runners", link: "/runners/claude-code" },
    ],

    sidebar: [
      {
        text: "Guide",
        items: [
          { text: "Getting Started", link: "/guide/getting-started" },
          { text: "Runner Interface", link: "/guide/runner-interface" },
          { text: "Channels", link: "/guide/channels" },
        ],
      },
      {
        text: "Libraries",
        items: [
          { text: "Go", link: "/libraries/go" },
          { text: "TypeScript", link: "/libraries/typescript" },
          { text: "Python", link: "/libraries/python" },
        ],
      },
      {
        text: "Runners",
        items: [
          { text: "Claude Code", link: "/runners/claude-code" },
          { text: "Ollama", link: "/runners/ollama" },
        ],
      },
    ],

    socialLinks: [
      { icon: "github", link: "https://github.com/anthropics/agentrunner" },
    ],
  },
});
