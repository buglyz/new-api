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
import type { QuotaDataItem } from '@/features/dashboard/types'

const SPARKLINE_BUCKETS = 12

export interface PersonalUsageMetrics {
  tokens: number
  requests: number
}

export interface PersonalUsageSparklines {
  tokens: number[]
  requests: number[]
}

export function formatTokenCount(
  value: string | number | null | undefined,
  locales?: Intl.LocalesArgument
): string {
  if (typeof value === 'number') {
    if (!Number.isSafeInteger(value) || value < 0) return '-'
    return Intl.NumberFormat(locales, { maximumFractionDigits: 0 }).format(
      value
    )
  }
  if (typeof value !== 'string' || !/^\d+$/.test(value)) return '-'

  return Intl.NumberFormat(locales, { maximumFractionDigits: 0 }).format(
    BigInt(value)
  )
}

export function formatTokenCountInMillions(
  value: string | number | null | undefined,
  locales?: Intl.LocalesArgument
): string {
  const tokenCount =
    typeof value === 'number'
      ? Number.isSafeInteger(value) && value >= 0
        ? BigInt(value)
        : null
      : typeof value === 'string' && /^\d+$/.test(value)
        ? BigInt(value)
        : null

  if (tokenCount === null) return '-'

  const million = 1_000_000n
  if (tokenCount < million) {
    return formatTokenCount(tokenCount.toString(), locales)
  }

  let wholeMillions = tokenCount / million
  let hundredths = Math.round(Number(tokenCount % million) / 10_000)
  if (hundredths === 100) {
    wholeMillions += 1n
    hundredths = 0
  }

  const fraction = hundredths
    ? `.${hundredths.toString().padStart(2, '0').replace(/0+$/, '')}`
    : ''
  return `${formatTokenCount(wholeMillions.toString(), locales)}${fraction}M`
}

export function calculatePersonalUsage(
  data: QuotaDataItem[]
): PersonalUsageMetrics {
  return data.reduce(
    (total, item) => ({
      tokens: total.tokens + (Number(item.token_used) || 0),
      requests: total.requests + (Number(item.count) || 0),
    }),
    { tokens: 0, requests: 0 }
  )
}

export function buildPersonalUsageSparklines(
  data: QuotaDataItem[],
  start: number,
  end: number
): PersonalUsageSparklines {
  const tokens = Array.from({ length: SPARKLINE_BUCKETS }, () => 0)
  const requests = Array.from({ length: SPARKLINE_BUCKETS }, () => 0)
  const duration = Math.max(1, end - start)

  for (const item of data) {
    const timestamp = Number(item.created_at) || start
    const ratio = (timestamp - start) / duration
    const index = Math.min(
      SPARKLINE_BUCKETS - 1,
      Math.max(0, Math.floor(ratio * SPARKLINE_BUCKETS))
    )
    tokens[index] += Number(item.token_used) || 0
    requests[index] += Number(item.count) || 0
  }

  return { tokens, requests }
}
