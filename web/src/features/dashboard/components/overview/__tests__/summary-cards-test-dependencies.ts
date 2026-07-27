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
let summaryApiFailure = false

async function loadHarnessDependencies() {
  const react = await import('react')
  const reactDom = await import('react-dom/client')
  const reactQuery = await import('@tanstack/react-query')
  const router = await import('@tanstack/react-router')
  const i18next = await import('i18next')
  const reactI18next = await import('react-i18next')
  const axios = await import('axios')
  const { api } = await import('@/lib/api')
  const { useAuthStore } = await import('@/stores/auth-store')
  const { PersonalUsageSummaryView } =
    await import('../personal-usage-summary-view')
  const { SummaryCards } = await import('../summary-cards')
  const i18n = i18next.createInstance()
  await i18n.use(reactI18next.initReactI18next).init({
    lng: 'en',
    resources: {},
  })
  const originalAdapter = api.defaults.adapter
  api.defaults.adapter = async (config) => {
    if (config.url === '/api/data/self/summary' && summaryApiFailure) {
      throw new Error('token summary unavailable')
    }
    return {
      data:
        config.url === '/api/data/self/summary'
          ? {
              success: true,
              data: {
                last_24h_tokens: '1200',
                total_tokens: '9007199254740993',
                tokens_tracked: true,
              },
            }
          : { success: true, data: [] },
      status: 200,
      statusText: 'OK',
      headers: new axios.AxiosHeaders(),
      config,
    }
  }
  return {
    ...react,
    ...reactDom,
    ...reactQuery,
    ...router,
    ...reactI18next,
    PersonalUsageSummaryView,
    SummaryCards,
    api,
    i18n,
    originalAdapter,
    useAuthStore,
  }
}

let dependencyPromise: ReturnType<typeof loadHarnessDependencies> | undefined

export function getHarnessDependencies() {
  return (dependencyPromise ??= loadHarnessDependencies())
}

export function setSummaryApiFailure(fail: boolean) {
  summaryApiFailure = fail
}

export async function restoreHarnessAdapter() {
  summaryApiFailure = false
  if (!dependencyPromise) return
  const dependencies = await dependencyPromise
  dependencies.api.defaults.adapter = dependencies.originalAdapter
}
