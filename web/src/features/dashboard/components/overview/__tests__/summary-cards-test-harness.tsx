/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { Window } from 'happy-dom'
import type React from 'react'

import {
  getHarnessDependencies,
  restoreHarnessAdapter,
} from './summary-cards-test-dependencies'

export { setSummaryApiFailure } from './summary-cards-test-dependencies'

export const domWindow = new Window({ url: 'http://localhost/' })
for (const key of [
  'window',
  'document',
  'navigator',
  'HTMLElement',
  'SVGElement',
  'Node',
  'Element',
  'Event',
  'CustomEvent',
  'MutationObserver',
  'requestAnimationFrame',
  'cancelAnimationFrame',
  'getComputedStyle',
] as const) {
  Object.defineProperty(globalThis, key, {
    configurable: true,
    value: domWindow[key],
  })
}
Object.assign(globalThis, {
  IS_REACT_ACT_ENVIRONMENT: true,
  scrollTo: () => undefined,
})

export type RenderedSummary = {
  container: HTMLDivElement
  root: import('react-dom/client').Root
  queryClient?: import('@tanstack/react-query').QueryClient
}

export async function renderPersonalUsageView(
  overrides: Partial<
    React.ComponentProps<
      typeof import('../personal-usage-summary-view').PersonalUsageSummaryView
    >
  > = {}
): Promise<RenderedSummary> {
  const dependencies = await getHarnessDependencies()
  const props: React.ComponentProps<
    typeof dependencies.PersonalUsageSummaryView
  > = {
    last24hTokens: '1200',
    totalTokens: '9007199254740993',
    requestCount: 42,
    tokensTracked: true,
    summaryLoading: false,
    summaryError: false,
    tokenSparkline: [1, 2, 3],
    requestSparkline: [3, 2, 1],
    ...overrides,
  }
  const container = document.createElement('div')
  document.body.append(container)
  const root = dependencies.createRoot(container)
  await dependencies.act(async () => {
    root.render(
      <dependencies.I18nextProvider i18n={dependencies.i18n}>
        <dependencies.PersonalUsageSummaryView {...props} />
      </dependencies.I18nextProvider>
    )
  })
  return { container, root }
}

export async function renderSummaryMode(
  personalMode: boolean
): Promise<RenderedSummary> {
  const dependencies = await getHarnessDependencies()
  const queryClient = new dependencies.QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  queryClient.setQueryData(['status'], {
    self_use_mode_enabled: personalMode,
    display_in_currency: true,
  })
  dependencies.useAuthStore.getState().auth.setUser({
    id: 1,
    username: 'owner',
    role: 100,
    quota: 500000,
    used_quota: 250000,
    request_count: 42,
  })
  const rootRoute = dependencies.createRootRoute({
    component: dependencies.Outlet,
  })
  const indexRoute = dependencies.createRoute({
    getParentRoute: () => rootRoute,
    path: '/',
    component: dependencies.SummaryCards,
  })
  const router = dependencies.createRouter({
    routeTree: rootRoute.addChildren([indexRoute]),
    history: dependencies.createMemoryHistory({ initialEntries: ['/'] }),
  })
  const container = document.createElement('div')
  document.body.append(container)
  const root = dependencies.createRoot(container)
  await dependencies.act(async () => {
    root.render(
      <dependencies.QueryClientProvider client={queryClient}>
        <dependencies.I18nextProvider i18n={dependencies.i18n}>
          <dependencies.RouterProvider router={router} />
        </dependencies.I18nextProvider>
      </dependencies.QueryClientProvider>
    )
    await router.load()
  })
  await dependencies.act(async () =>
    queryClient.refetchQueries({
      predicate: (query) => query.queryKey[0] === 'dashboard',
    })
  )
  return { container, root, queryClient }
}

export async function unmountSummary(rendered: RenderedSummary) {
  const dependencies = await getHarnessDependencies()
  await dependencies.act(async () => rendered.queryClient?.cancelQueries())
  await dependencies.act(async () => rendered.root.unmount())
  rendered.queryClient?.clear()
  rendered.container.remove()
  dependencies.useAuthStore.getState().auth.reset()
}

export async function closeSummaryHarness() {
  await restoreHarnessAdapter()
  domWindow.close()
}
