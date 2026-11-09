This is a Vite + React web app (TypeScript, react-router-dom, @tanstack/react-query, zustand). No React Native, no Expo -- it's a plain browser SPA.

## Commands

```bash
npm run dev         # start the Vite dev server (proxies /api to the Argus server on :9100 -- see vite.config.ts)
npm run build:web   # production build, written into ../internal/webui/dist for cmd/argus to embed
npm run typecheck    # tsc --noEmit
npm run lint         # eslint .
npm test             # vitest run
```

Run typecheck, lint, and tests before declaring any task done.

## Structure

- `src/routes/`: one file per screen/route. The route tree itself is defined explicitly in `src/router.tsx` (plain `react-router-dom` `<Routes>`, not file-based routing).
- `src/auth/`: zustand session store (`session.ts`) plus `ProtectedRoute`/`PublicRoute` wrapper components that gate routes on sign-in status.
- `src/api/`: `openapi-fetch` client; `schema.d.ts` is generated (`npm run gen:api`) from `../internal/api/openapi/openapi.yaml` -- don't hand-edit it.
- `src/components/`, `src/hooks/`, `src/lib/`, `src/theme/`: plain React components/hooks/logic, no routing concerns.
- `src/test/setup.ts`: vitest setup (jsdom polyfills for `matchMedia`/`ResizeObserver`, jest-dom matchers).

## Conventions

- Styling is plain inline `style` objects (React's `CSSProperties`), no CSS Modules/Tailwind/CSS-in-JS. `src/index.css` holds only what inline styles can't do (resets, `:focus-visible`, `@keyframes`).
- Small pure helpers are colocated with the component that uses them (e.g. `charts/BarList.tsx` exports both `BarList` and `barItems`) for testability -- this is intentional, not an oversight (`react-refresh/only-export-components` is disabled in `eslint.config.js` for this reason).
- Virtualized lists (live tail, log history) use `components/VirtualList.tsx` (`@tanstack/react-virtual`), not a third-party list component.
