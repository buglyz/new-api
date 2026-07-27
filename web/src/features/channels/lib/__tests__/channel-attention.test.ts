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
import { getChannelAttentionReasons } from '../channel-attention'

const NOW = 2_000_000

function channel(overrides: Partial<Channel> = {}): Channel {
  return {
    id: 1,
    type: 1,
    key: '',
    status: 1,
    name: 'upstream',
    created_time: NOW - 100,
    test_time: NOW - 60,
    response_time: 500,
    balance: 0,
    balance_updated_time: 0,
    models: 'gpt-test',
    group: 'default',
    used_quota: 0,
    other: '',
    other_info: '',
    remark: '',
    max_input_tokens: 0,
    channel_info: {
      is_multi_key: false,
      multi_key_size: 0,
      multi_key_polling_index: 0,
      multi_key_mode: 'random',
    },
    settings: '{}',
    ...overrides,
  }
}

describe('channel attention classification', () => {
  test('keeps a recently probed responsive channel out of attention', () => {
    assert.deepEqual(getChannelAttentionReasons(channel(), NOW), [])
  })

  test('classifies disabled, untested, stale, and slow probe states', () => {
    assert.deepEqual(
      getChannelAttentionReasons(
        channel({ status: 3, test_time: 0, response_time: 5_001 }),
        NOW
      ),
      ['auto_disabled', 'untested', 'slow_probe']
    )

    assert.deepEqual(
      getChannelAttentionReasons(
        channel({ test_time: NOW - 24 * 60 * 60 - 1 }),
        NOW
      ),
      ['stale_probe']
    )
  })

  test('classifies a multi-key channel only when every key is unavailable', () => {
    const baseInfo = {
      is_multi_key: true,
      multi_key_size: 2,
      multi_key_polling_index: 0,
      multi_key_mode: 'random' as const,
    }
    assert.deepEqual(
      getChannelAttentionReasons(
        channel({
          channel_info: {
            ...baseInfo,
            multi_key_status_list: { '0': 3 },
          },
        }),
        NOW
      ),
      []
    )
    assert.deepEqual(
      getChannelAttentionReasons(
        channel({
          channel_info: {
            ...baseInfo,
            multi_key_status_list: { '0': 3, '1': 2 },
          },
        }),
        NOW
      ),
      ['all_keys_unavailable']
    )
  })
})
