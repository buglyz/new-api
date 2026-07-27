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
interface PageResponse<T> {
  success: boolean
  message?: string
  data?: {
    items: T[]
    total: number
  }
}

interface FetchAllPagesOptions<T> {
  fetchPage: (page: number, pageSize: number) => Promise<PageResponse<T>>
  getId: (item: T) => number
  loadError: string
  incompleteError: string
  pageSize?: number
}

export async function fetchAllPages<T>(
  options: FetchAllPagesOptions<T>
): Promise<T[]> {
  const itemsById = new Map<number, T>()
  const pageSize = options.pageSize ?? 100
  let page = 1
  let total = Number.POSITIVE_INFINITY

  while (itemsById.size < total) {
    const result = await options.fetchPage(page, pageSize)
    if (!result.success || !result.data) {
      throw new Error(result.message || options.loadError)
    }

    const reportedTotal = result.data.total
    if (
      !Array.isArray(result.data.items) ||
      !Number.isInteger(reportedTotal) ||
      reportedTotal < 0
    ) {
      throw new Error(options.incompleteError)
    }

    total = reportedTotal
    for (const item of result.data.items) {
      itemsById.set(options.getId(item), item)
    }

    if (result.data.items.length < pageSize) break
    page += 1
  }

  if (itemsById.size < total) throw new Error(options.incompleteError)
  return [...itemsById.values()]
}
