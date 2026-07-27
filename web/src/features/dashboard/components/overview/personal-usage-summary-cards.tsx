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
import { useQuery } from '@tanstack/react-query'
import { useMemo } from 'react'

import {
  getPersonalUsageSummary,
  getUserQuotaDates,
} from '@/features/dashboard/api'
import { buildPersonalUsageSparklines } from '@/features/dashboard/lib/personal-usage-summary'
import { computeTimeRange } from '@/lib/time'
import { useAuthStore } from '@/stores/auth-store'

import { PersonalUsageSummaryView } from './personal-usage-summary-view'

export function PersonalUsageSummaryCards() {
  const requestCount = useAuthStore(
    (state) => state.auth.user?.request_count ?? 0
  )
  const timeRange = useMemo(() => computeTimeRange(1), [])
  const trendQuery = useQuery({
    queryKey: [
      'dashboard',
      'overview',
      'personal-token-trend',
      timeRange.start_timestamp,
      timeRange.end_timestamp,
    ],
    queryFn: () => getUserQuotaDates(timeRange),
    staleTime: 60 * 1000,
  })
  const summaryQuery = useQuery({
    queryKey: ['dashboard', 'overview', 'personal-token-summary'],
    queryFn: getPersonalUsageSummary,
    staleTime: 60 * 1000,
  })
  const trendData = useMemo(
    () => trendQuery.data?.data ?? [],
    [trendQuery.data?.data]
  )
  const sparklines = useMemo(
    () =>
      buildPersonalUsageSparklines(
        trendData,
        timeRange.start_timestamp,
        timeRange.end_timestamp
      ),
    [timeRange.end_timestamp, timeRange.start_timestamp, trendData]
  )

  return (
    <PersonalUsageSummaryView
      last24hTokens={summaryQuery.data?.data.last_24h_tokens}
      totalTokens={summaryQuery.data?.data.total_tokens}
      requestCount={requestCount}
      tokensTracked={summaryQuery.data?.data.tokens_tracked ?? true}
      summaryLoading={summaryQuery.isPending}
      summaryError={summaryQuery.isError}
      tokenSparkline={trendQuery.isError ? [] : sparklines.tokens}
      requestSparkline={trendQuery.isError ? [] : sparklines.requests}
    />
  )
}
