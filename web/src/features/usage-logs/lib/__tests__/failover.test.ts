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

import type { UsageLog } from '../../data/schema'
import {
  getFailoverTrace,
  getObservedFailovers,
  getRelatedLogSearch,
} from '../failover'

function log(overrides: Partial<UsageLog> = {}): UsageLog {
  return {
    id: 1,
    user_id: 1,
    created_at: 2_000_000,
    type: 2,
    content: '',
    username: '',
    token_name: '',
    model_name: 'gpt-test',
    quota: 0,
    prompt_tokens: 0,
    completion_tokens: 0,
    use_time: 1,
    is_stream: false,
    channel: 3,
    channel_name: 'final',
    token_id: 1,
    group: 'default',
    ip: '',
    other: JSON.stringify({ admin_info: { use_channel: ['1', '2', '3'] } }),
    request_id: 'req-123',
    upstream_request_id: '',
    ...overrides,
  }
}

describe('failover trace', () => {
  test('exposes attempts, retry count, final success, and request ID', () => {
    assert.deepEqual(getFailoverTrace(log()), {
      attemptChannelIds: [1, 2, 3],
      retryCount: 2,
      finalSuccessfulChannelId: 3,
      requestId: 'req-123',
    })
  })

  test('does not claim a final successful channel for an error log', () => {
    assert.equal(
      getFailoverTrace(log({ type: 5 }))?.finalSuccessfulChannelId,
      null
    )
  })

  test('ignores a single channel because no retry occurred', () => {
    assert.equal(
      getFailoverTrace(
        log({ other: JSON.stringify({ admin_info: { use_channel: ['3'] } }) })
      ),
      null
    )
  })

  test('keeps the request ID search within the original log time window', () => {
    assert.deepEqual(getRelatedLogSearch(log()), {
      page: 1,
      requestId: 'req-123',
      startTime: 1_999_700_000,
      endTime: 2_000_300_000,
    })
  })

  test('counts each request once while retaining all observed failovers', () => {
    const logs = [
      log({ id: 5, request_id: 'req-4' }),
      log({ id: 4, request_id: 'req-3' }),
      log({ id: 3, request_id: 'req-2' }),
      log({ id: 2, request_id: 'req-1' }),
      log({ id: 1, request_id: 'req-1', type: 5 }),
    ]

    const observed = getObservedFailovers(logs)

    assert.equal(observed.length, 4)
    assert.deepEqual(
      observed.map((item) => item.log.id),
      [5, 4, 3, 2]
    )
  })
})
