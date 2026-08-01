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

import type { Channel } from '../../types'
import { aggregateChannelsByTag, isTagAggregateRow } from '../channel-utils'

function channel(id: number, requestCount: number): Channel {
  return {
    id,
    tag: 'shared',
    group: 'default',
    status: 1,
    used_quota: 0,
    request_count: requestCount,
    response_time: 0,
    priority: 0,
    weight: 0,
  } as Channel
}

describe('channel request count display data', () => {
  test('sums request counts on a tag aggregate row', () => {
    const rows = aggregateChannelsByTag([channel(1, 3), channel(2, 5)])
    const tagRow = rows[0]

    assert.equal(isTagAggregateRow(tagRow), true)
    if (isTagAggregateRow(tagRow)) {
      assert.equal(tagRow.request_count, 8)
    }
  })
})
