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

import type { ChannelMonitorTarget } from '../../types'
import { groupChannelMonitorTargets } from '../channel-monitor-groups'

function target(
  overrides: Partial<ChannelMonitorTarget>
): ChannelMonitorTarget {
  return {
    channel_id: 1,
    channel_name: 'primary',
    groups: 'default',
    model: 'gpt-test',
    status: 'success',
    health: 'healthy',
    state_changed: false,
    attempts: 1,
    latency_ms: 100,
    http_status: 200,
    error: '',
    created_at: 100,
    success_rate_24h: 1,
    samples_24h: 2,
    ...overrides,
  }
}

describe('channel monitor grouping', () => {
  test('combines model samples into a weighted channel success rate', () => {
    const groups = groupChannelMonitorTargets([
      target({ model: 'model-b', success_rate_24h: 0.5, samples_24h: 4 }),
      target({ model: 'model-a', success_rate_24h: 1, samples_24h: 2 }),
    ])

    assert.equal(groups.length, 1)
    assert.equal(groups[0]?.samples_24h, 6)
    assert.equal(groups[0]?.success_rate_24h, 2 / 3)
    assert.deepEqual(
      groups[0]?.targets.map((item) => item.model),
      ['model-a', 'model-b']
    )
  })

  test('keeps channels separate and promotes the worst health state', () => {
    const groups = groupChannelMonitorTargets([
      target({ channel_id: 2, channel_name: 'backup', health: 'down' }),
      target({ channel_id: 1, health: 'degraded' }),
    ])

    assert.deepEqual(
      groups.map((group) => [group.channel_id, group.health]),
      [
        [1, 'degraded'],
        [2, 'down'],
      ]
    )
  })
})
