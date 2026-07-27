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
import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import { fetchAllPages } from '../fetch-all-pages'

describe('full pagination contract', () => {
  test('loads every page until the reported total is covered', async () => {
    const calls: Array<[number, number]> = []
    const result = await fetchAllPages({
      fetchPage: async (page, pageSize) => {
        calls.push([page, pageSize])
        const start = (page - 1) * pageSize
        const length = page === 1 ? 100 : 1
        return {
          success: true,
          data: {
            items: Array.from({ length }, (_, index) => ({
              id: start + index + 1,
            })),
            total: 101,
          },
        }
      },
      getId: (item) => item.id,
      loadError: 'load failed',
      incompleteError: 'pagination incomplete',
    })

    assert.equal(result.length, 101)
    assert.deepEqual(calls, [
      [1, 100],
      [2, 100],
    ])
  })

  test('rejects an incomplete page instead of silently undercounting', async () => {
    await assert.rejects(
      fetchAllPages({
        fetchPage: async () => ({
          success: true,
          data: { items: [{ id: 1 }], total: 2 },
        }),
        getId: (item) => item.id,
        loadError: 'load failed',
        incompleteError: 'pagination incomplete',
      }),
      /pagination incomplete/
    )
  })

  test('rejects an invalid reported total as a contract failure', async () => {
    await assert.rejects(
      fetchAllPages({
        fetchPage: async () => ({
          success: true,
          data: { items: [], total: Number.NaN },
        }),
        getId: (item: { id: number }) => item.id,
        loadError: 'load failed',
        incompleteError: 'pagination incomplete',
      }),
      /pagination incomplete/
    )
  })
})
