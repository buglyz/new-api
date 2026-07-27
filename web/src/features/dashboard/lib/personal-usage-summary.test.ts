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

import {
  buildPersonalUsageSparklines,
  calculatePersonalUsage,
  formatTokenCount,
} from './personal-usage-summary'

describe('personal usage summary', () => {
  test('aggregates token usage independently from quota', () => {
    assert.deepEqual(
      calculatePersonalUsage([
        { created_at: 100, token_used: 120, count: 2, quota: 9_999 },
        { created_at: 200, token_used: 80, count: 1, quota: 1 },
      ]),
      { tokens: 200, requests: 3 }
    )
  })

  test('builds bounded token and request sparklines', () => {
    const result = buildPersonalUsageSparklines(
      [
        { created_at: 0, token_used: 10, count: 1 },
        { created_at: 150, token_used: 20, count: 2 },
        { created_at: 500, token_used: 30, count: 3 },
      ],
      100,
      200
    )

    assert.equal(result.tokens[0], 10)
    assert.equal(result.tokens[6], 20)
    assert.equal(result.tokens[11], 30)
    assert.equal(
      result.requests.reduce((sum, value) => sum + value, 0),
      6
    )
  })

  test('formats exact token totals beyond the JavaScript safe integer range', () => {
    assert.equal(
      formatTokenCount('9007199254740993', 'en-US'),
      '9,007,199,254,740,993'
    )
    assert.equal(formatTokenCount('0', 'en-US'), '0')
    assert.equal(formatTokenCount('invalid', 'en-US'), '-')
  })
})
