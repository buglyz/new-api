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
import { describe, expect, it } from 'vitest'

import {
  buildPersonalUsageSparklines,
  calculatePersonalUsage,
} from './personal-usage-summary'

describe('personal usage summary', () => {
  it('aggregates token usage independently from quota', () => {
    expect(
      calculatePersonalUsage([
        { created_at: 100, token_used: 120, count: 2, quota: 9_999 },
        { created_at: 200, token_used: 80, count: 1, quota: 1 },
      ])
    ).toEqual({ tokens: 200, requests: 3 })
  })

  it('builds bounded token and request sparklines', () => {
    const result = buildPersonalUsageSparklines(
      [
        { created_at: 0, token_used: 10, count: 1 },
        { created_at: 150, token_used: 20, count: 2 },
        { created_at: 500, token_used: 30, count: 3 },
      ],
      100,
      200
    )

    expect(result.tokens[0]).toBe(10)
    expect(result.tokens[6]).toBe(20)
    expect(result.tokens[11]).toBe(30)
    expect(result.requests.reduce((sum, value) => sum + value, 0)).toBe(6)
  })
})
