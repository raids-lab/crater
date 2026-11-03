[English](CONTRIBUTING.md) | [简体中文](CONTRIBUTING.zh-CN.md)

# Contributing to Crater Frontend

This document is the complete specification for developing `frontend/`. Read the global rules and contribution workflow in the repository root [CONTRIBUTING.md](../CONTRIBUTING.md) first, then develop per this document.

Use this file when you change React routes, components, hooks, frontend API calls, i18n strings, UI behavior, or frontend build/lint workflow.

## Tech Stack

- **Language**: TypeScript
- **Framework**: React 19
- **Routing**: TanStack Router
- **Data fetching**: TanStack Query v5
- **State management**: Jotai
- **Styling**: Tailwind CSS
- **UI libraries**: shadcn/ui (headless), Flowbite (Tailwind templates), TanStack Table (headless tables)

## Local Debugging

- Install Node.js and pnpm (use [nvm](https://github.com/nvm-sh/nvm) to manage Node versions): `node -v` should be v22+, `pnpm -v` should be v10+.
- VS Code users can import `.vscode/React.code-profile` via `Profiles > Import Profile` and install recommended extensions; for other IDEs, configure Prettier, ESLint, and Tailwind CSS IntelliSense manually.

For most frontend work, debug with the frontend and backend running together. The backend can connect to existing test-cluster dependencies through its config, so do not try to start the whole Crater cluster or local databases for ordinary UI work. Use MSW only when API mocking is appropriate for the task.

The frontend dev server proxies main backend APIs through `VITE_SERVER_PROXY_BACKEND` to `http://localhost:8088/` by default, and storage APIs through `VITE_SERVER_PROXY_STORAGE` to `http://localhost:7320/` by default. Start the backend storage server with `cd ../backend && make run-storage` when testing storage-related pages or flows such as file management, datasets, models, uploads, or downloads.

`make run` reads the dev server port only from an explicit `PORT=...` entry in `.env.development`. If that entry is missing, it falls back to `5180`. Proxy variables such as `VITE_SERVER_PROXY_BACKEND` and `VITE_SERVER_PROXY_STORAGE` are not treated as port settings.

```bash
cd frontend
pnpm install
make run
```

Per the global rules, build/lint go through `make` and should be **run locally by the maintainer**; output the command and reason when verification is needed. For auto-fixable lint / formatting issues, prefer `make lint-fix` or the developer's usual `pnpm lint --fix`; use `make format` for full Prettier formatting and `make format-translation` for locale file formatting.

### API Mocking (MSW)

For development you can mock APIs with MSW: set `VITE_USE_MSW=true` in `.env.development` and add handlers in `src/mocks/handlers.ts`. Prefer managing `.env.development` via the unified configuration management at the repo root (see the root CONTRIBUTING).

### Dependency Management

```bash
pnpm outdated            # check for updates
pnpm update              # minor updates
pnpm update --latest     # major updates (use cautiously)
```

Update shadcn components:

```bash
for file in src/components/ui/*.tsx; do
  pnpm dlx shadcn@latest add -yo $(basename "$file" .tsx)
done
```

## Project Structure

```
src/
├── components/           # Reusable components
│   ├── ui/               # shadcn components
│   ├── ui-custom/        # Custom styling-layer components
│   ├── custom/           # Custom business components
│   └── layout/           # Layouts
├── hooks/                # Custom hooks
├── services/             # API services
├── routes/               # Route-based pages
└── ...
```

## Component Reuse (Core Requirements)

- Prefer reusing existing UI, business, form, and hook building blocks before creating new ones. Check `src/components/ui-custom/` (styling layer), `src/components/form/` (form controls and metadata forms), `src/components/` (business components), and `src/hooks/` first.
- Only create a new component, form control, or hook when existing building blocks do not fit the behavior, layout, or domain model. Keep the new abstraction close to the feature unless there is clear reuse demand.
- **Modifying widely-referenced shared components requires great caution**: assess risk, check all references, and explain a solid reason to the maintainer before changing. For highly-reused base components, form controls, metadata forms, hooks, or anything under `ui-custom/`, explicitly confirm the impact scope and compatibility on referencing pages, then ask the developer to manually spot-check representative affected pages.

## Hooks (Core Requirements)

- Use `useIsAdmin()` from `src/hooks/use-admin.tsx` to determine the current identity; do not reimplement it.
- Before implementing a feature, check `src/hooks/` for a directly reusable hook.

## APIs and Errors

- Admin views call admin APIs (with the `admin` prefix); regular users call user APIs (no prefix), corresponding to the backend routes.
- API errors use the backend response envelope `code`, `data`, and `msg`. The default UI should preserve the backend `msg` when it helps the user understand what failed, and include stable troubleshooting facts such as HTTP status and business code when the existing error component supports them.
- Treat `src/services/error_code.ts` as the generated frontend reference for business error codes and groups. Its source is the backend definitions parsed by `src/services/generator.py`, primarily `backend/internal/bizerr/groups.go`. Do not copy numeric code tables into UI code or docs; import the generated constants and use specific codes only as examples in prose.
- The frontend **does not interpret business error codes by default**. Without a logical need for special handling, use the shared error handling path (`showErrorToast`, `handleApiErrorByCode`, `markApiErrorHandled`) instead of replacing the backend message with a generic page-local toast.
- Add business-code-specific handling only when the page must change behavior, such as highlighting a field, showing a conflict-resolution flow, suppressing an auth retry message, or guiding an administrator to inspect a dependency. In that case, handle the narrowest generated code from `src/services/error_code.ts`, call `markApiErrorHandled` when the error is consumed, and keep a fallback that still shows the backend `msg`.
- User-facing error text should answer what failed and what the user can do next. Administrator-facing flows should also keep enough facts for troubleshooting, such as the affected resource, HTTP status, business code, backend message, and any safe correlation or object identifier already present in the page.
- Do not swallow errors in `catch` / `onError`, log them only to the console, or replace actionable backend messages with `operation failed` style text. If the backend message is not safe or not actionable, fix the backend error contract instead of hiding it in the frontend.
- Non-idempotent operations (create, update, delete, stop, lock/unlock, quota changes, etc.) must show a confirmation dialog before execution. The dialog should clearly name the target and consequence, and should reuse the project's existing dialog / alert-dialog patterns.
- For potentially long-running non-idempotent requests, also add a loading state and disable the relevant buttons to prevent duplicate submission.
- Job creation, clone-job, and job-template flows depend on the versioned template JSON produced by `MetadataForm*` in `src/components/form/types.ts` (`version`, `type`, `data`) and parsed by the template import helpers in `src/utils/form.ts`. When changing persisted job configuration fields or form data shape, decide whether the relevant `MetadataForm*` version must be bumped. If old templates should remain usable, register a migration from the previous version to the new version; later old versions must migrate by chaining adjacent-version migrations. Unsupported older versions must fail clearly instead of being parsed as current data. Any new loader for job-template data, clone source, or template source must reuse the shared template import/migration helpers; do not hand-roll per-entry compatibility, and do not apply job-template migration to arbitrary JSON that is not a job-template payload.

## Internationalization (Core Requirements)

- **No hardcoded text**; integrate the project's i18n solution and write labels in Chinese only at first.
- Translation keys must be English semantic keys, not Chinese source text. Follow the current dotted-key style and place new keys under the appropriate domain by feature, page, or component, such as `navigation.*`, `jobs.*`, `accountDetail.*`, `accountForm.*`, or `adminJobOverview.*`.
- Do not add new Chinese translation keys. Some existing locale files still contain Chinese keys as historical debt; leave broad migration to a dedicated task unless the current change explicitly touches those strings.
- When adding or changing translatable text, update every locale `translation.json` in the same change. Keep translations accurate and keep project-specific terminology consistent across languages.

## What's New

- Write a short version heading, an introduction summarizing progress since the previous release, and a few thematic highlights. Explain concrete user-visible changes and benefits; do not dump commits, internal implementation details, or unsupported claims. Derive the facts from changes included in the release; clearly distinguish any future work from available features.
- Update `CURRENT_VERSION` in `src/components/layout/what-is-new.tsx` together with `whatsNew.content` in every locale. Keep the `{{version}}` interpolation and equivalent meaning across languages. This version controls whether users have acknowledged the dialog; it is not the application build version or an API compatibility counter.
- Apply the existing frontend verification rules: check the rendered Markdown and translations in the actual browser, verify that a new version opens for a user who acknowledged the previous version, and that confirming it prevents repeated automatic opening for that version. Record automated checks and developer checks separately.
- The in-app text may end with a brief invitation to share feedback. When reusing it for GitHub Release, follow the root [Release Notes](../CONTRIBUTING.md#release-notes) rules instead of copying the dialog verbatim.

## Experience and Consistency

- Keep new pages consistent with existing pages in layout, style, and colors (reference existing page layouts).
- Do not forget to set breadcrumbs for new pages.
- For inputs, switches, or configuration items that may be hard to understand, add a small help icon next to the label/title with a hover tooltip explaining what it does, when to use it, and any key mechanism or consequence. Do not assume platform users or admins already understand cloud-computing, Kubernetes, scheduling, storage, networking, or system-domain terms.
- Mind human-centered details (e.g. button order), and add responsive adaptation for narrow screens when appropriate.

## Before Submitting Frontend Changes

Follow the Commit convention in the root CONTRIBUTING (`type(scope): subject`, scope e.g. `portal`, `admin`, `ui`, `api`). When reporting bugs, include reproduction steps, expected vs actual behavior, screenshots (if any), and browser/OS version.

For frontend / UI changes, provide actual screenshots of the affected pages, roles, and states in the PR description or a comment, following the root CONTRIBUTING screenshot rules; explain when there is no meaningful visible change or capture is unavailable. Screenshots from an Agent's real execution belong under AI checks, not developer manual verification; AI-generated or simulated images are not evidence. Missing or incomplete screenshots warrant only a brief review-overview reminder, not a code finding or a merge-blocking recommendation solely on that basis.

Before opening or updating a PR:

- Run the relevant `make` checks, usually `make pre-commit-check`.
- For changed API error paths, verify both the generic error presentation and any business-code-specific handling, including the message users see and the facts administrators can use for troubleshooting.
- Confirm all visible text uses i18n and all locale `translation.json` files are synchronized.
- Confirm new or changed translation keys are English semantic keys in the right domain.
- Confirm affected pages were manually checked by the developer, with role, page, action, and observed result recorded for the PR.
- Provide frontend / UI screenshots or explain why they are unavailable, as described above.

## Known Issues

- Dark-mode input styling: browser autofill produces white backgrounds in dark mode ([TailwindCSS#8679](https://github.com/tailwindlabs/tailwindcss/discussions/8679)).
