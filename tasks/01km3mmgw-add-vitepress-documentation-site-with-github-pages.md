---
title: "Add VitePress documentation site with GitHub Pages deployment"
id: "01km3mmgw"
status: pending
priority: medium
type: feature
tags: ["docs", "vitepress"]
created: "2026-03-19"
---

# Add VitePress documentation site with GitHub Pages deployment

## Objective

Set up a VitePress documentation site for the agentrunner monorepo and configure GitHub Pages for automated deployment. The docs should cover the project overview, getting started guide, per-language library usage (Go, TypeScript, Python, Java), the common Runner interface, and example programs.

## Tasks

- [ ] Initialize VitePress project under `docs/` directory (`npm init` + `vitepress` dependency)
- [ ] Configure `docs/.vitepress/config.ts` with site title, description, nav, and sidebar structure
- [ ] Write landing page (`docs/index.md`) with project overview and quickstart links
- [ ] Write "Getting Started" guide covering installation and basic usage
- [ ] Write "Runner Interface" page documenting the common interface from `INTERFACE.md`
- [ ] Write per-language library pages: Go, TypeScript, Python, Java
- [ ] Write "Runners" section with pages for each supported runner (Claude Code, Gemini, Codex, Ollama)
- [ ] Add npm scripts for `docs:dev`, `docs:build`, and `docs:preview`
- [ ] Add GitHub Actions workflow (`.github/workflows/docs.yml`) to build and deploy to GitHub Pages on push to `main`
- [ ] Configure GitHub Pages source to use GitHub Actions deployment
- [ ] Add `docs` targets to root Makefile (`docs-dev`, `docs-build`)
- [ ] Verify the site builds without errors and deploys successfully

## Acceptance Criteria

- `npm run docs:dev` starts local VitePress dev server from `docs/` directory
- `npm run docs:build` produces a static site under `docs/.vitepress/dist/`
- GitHub Actions workflow builds and deploys docs to GitHub Pages on push to `main`
- Documentation covers: project overview, getting started, runner interface, all language libraries, all runner types
- Site navigation includes sidebar and top nav for easy discovery
