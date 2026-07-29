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

import type { ApiKey } from '../../types'
import { getApiKeyAttentionReasons } from '../api-key-attention'

const NOW = 20_000_000

function apiKey(overrides: Partial<ApiKey> = {}): ApiKey {
  return {
    id: 1,
    name: 'client',
    key: 'abcd**********wxyz',
    status: 1,
    expired_time: NOW + 30 * 24 * 60 * 60,
    created_time: NOW - 100,
    accessed_time: NOW - 100,
    group: 'default',
    cross_group_retry: false,
    model_limits_enabled: true,
    model_limits: 'gpt-test',
    allow_ips: '',
    ...overrides,
  }
}

describe('API key attention classification', () => {
  test('keeps a bounded, active key out of attention', () => {
    assert.deepEqual(getApiKeyAttentionReasons(apiKey(), NOW), [])
  })

  test('classifies expiry, inactivity, and missing restrictions', () => {
    assert.deepEqual(
      getApiKeyAttentionReasons(
        apiKey({
          expired_time: NOW + 7 * 24 * 60 * 60,
          accessed_time: NOW - 90 * 24 * 60 * 60 - 1,
          model_limits_enabled: false,
        }),
        NOW
      ),
      ['expiring_soon', 'long_unused', 'no_model_limits']
    )
  })

  test('does not flag an active non-expiring key', () => {
    assert.deepEqual(getApiKeyAttentionReasons(apiKey({ expired_time: -1 }), NOW), [])
  })
})
