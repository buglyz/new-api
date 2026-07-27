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
import { describe, test } from 'bun:test'
import assert from 'node:assert/strict'

import { usageLogSchema } from '../../data/schema'
import type { LogOtherData } from '../../types'
import { buildTypeDetailSegments } from '../columns/common-logs-columns'

const t = (key: string) => key
const violationLog = usageLogSchema.parse({
  id: 1,
  user_id: 1,
  created_at: 1,
  type: 2,
  content: '',
  quota: 12_500,
})
const violationDetails: LogOtherData = {
  violation_fee: true,
  violation_fee_code: 'content_policy',
  fee_quota: 12_500,
}

describe('personal usage log financial copy', () => {
  test('hides violation fees while preserving the diagnostic code', () => {
    const segments = buildTypeDetailSegments(
      violationLog,
      violationDetails,
      t,
      true
    )

    assert.deepEqual(
      segments.map((segment) => segment.text),
      ['Policy violation', 'content_policy']
    )
  })

  test('preserves the standard violation fee summary', () => {
    const segments = buildTypeDetailSegments(
      violationLog,
      violationDetails,
      t,
      false
    )

    assert.equal(segments[0]?.text, 'Violation Fee')
    assert.match(segments.at(-1)?.text ?? '', /^Fee:/)
  })
})
