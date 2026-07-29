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
import { afterAll, describe, test } from 'bun:test'
import assert from 'node:assert/strict'

import {
  closeSummaryHarness,
  renderPersonalUsageView,
  renderSummaryMode,
  setSummaryApiFailure,
  unmountSummary,
} from './summary-cards-test-harness'

describe('summary cards mode and states', () => {
  afterAll(closeSummaryHarness)

  test('personal mode replaces all standard balance and wallet content', async () => {
    const rendered = await renderSummaryMode(true)
    const text = rendered.container.textContent ?? ''

    assert.match(text, /Last 24h tokens/)
    assert.match(text, /9,007,199,254,740,993/)
    for (const term of ['Credit remaining', 'Runway', 'Wallet', '$']) {
      assert.equal(text.includes(term), false, term)
    }
    await unmountSummary(rendered)
  })

  test('ignores legacy status and keeps the personal usage summary', async () => {
    const rendered = await renderSummaryMode(false)
    const text = rendered.container.textContent ?? ''

    assert.match(text, /Last 24h tokens/)
    for (const term of ['Credit remaining', 'Runway', 'Wallet', '$']) {
      assert.equal(text.includes(term), false, term)
    }
    await unmountSummary(rendered)
  })

  test('shows exact values without commercial labels on narrow layouts', async () => {
    const rendered = await renderPersonalUsageView()
    const text = rendered.container.textContent ?? ''

    assert.match(text, /1,200/)
    assert.match(text, /9,007,199,254,740,993/)
    assert.match(text, /42/)
    for (const term of ['Balance', 'Runway', 'Wallet', '$']) {
      assert.equal(text.includes(term), false, term)
    }
    const grid = rendered.container.querySelector(
      '[data-personal-usage-summary="true"] header + div'
    )
    assert.match(grid?.className ?? '', /grid-cols-1/)
    assert.match(grid?.className ?? '', /sm:grid-cols-3/)
    assert.match(grid?.className ?? '', /auto-rows-fr/)
    await unmountSummary(rendered)
  })

  test('renders zero and explains when usage tracking is disabled', async () => {
    const rendered = await renderPersonalUsageView({
      last24hTokens: '0',
      totalTokens: '0',
      tokensTracked: false,
    })

    assert.match(
      rendered.container.textContent ?? '',
      /Usage tracking is disabled; totals are historical/
    )
    assert.equal(
      [...rendered.container.querySelectorAll('.tabular-nums')].some(
        (node) => node.textContent === '0'
      ),
      true
    )
    await unmountSummary(rendered)
  })

  test('isolates a token API error from the request count', async () => {
    setSummaryApiFailure(true)
    const rendered = await renderSummaryMode(true)
    const values = [...rendered.container.querySelectorAll('.tabular-nums')]
      .map((node) => node.textContent)
      .filter(Boolean)

    assert.equal(values.filter((value) => value === '--').length, 2)
    assert.equal(values.includes('42'), true)
    await unmountSummary(rendered)
    setSummaryApiFailure(false)
  })
})
