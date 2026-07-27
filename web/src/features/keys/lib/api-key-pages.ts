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
import { fetchAllPages } from '@/lib/fetch-all-pages'

import { getApiKeys, searchApiKeys } from '../api'
import type { ApiKey } from '../types'

export async function fetchAllApiKeyPages(params: {
  keyword?: string
  token?: string
}): Promise<ApiKey[]> {
  const shouldSearch = Boolean(params.keyword?.trim() || params.token?.trim())
  return fetchAllPages({
    fetchPage: (page, pageSize) =>
      shouldSearch
        ? searchApiKeys({ ...params, p: page, size: pageSize })
        : getApiKeys({ p: page, size: pageSize }),
    getId: (apiKey) => apiKey.id,
    loadError: 'Failed to load all API keys',
    incompleteError: 'API key pagination ended before the reported total',
  })
}
