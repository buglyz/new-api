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
import { Activity, Flame, Layers } from 'lucide-react'
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'

import { StaggerContainer, StaggerItem } from '@/components/page-transition'
import {
  getPersonalUsageSummary,
  getUserQuotaDates,
} from '@/features/dashboard/api'
import {
  buildPersonalUsageSparklines,
  calculatePersonalUsage,
} from '@/features/dashboard/lib/personal-usage-summary'
import { useStatus } from '@/hooks/use-status'
import { formatNumber } from '@/lib/format'
import { computeTimeRange } from '@/lib/time'
import { useAuthStore } from '@/stores/auth-store'

import { StatCard } from '../ui/stat-card'

export function PersonalUsageSummaryCards() {
  const { t } = useTranslation()
  const requestCount = useAuthStore(
    (state) => state.auth.user?.request_count ?? 0
  )
  const { loading: statusLoading } = useStatus()
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
  const recentUsage = useMemo(
    () => calculatePersonalUsage(trendData),
    [trendData]
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
  const loading =
    statusLoading || trendQuery.isPending || summaryQuery.isPending
  const items = [
    {
      key: 'recentTokens',
      title: t('Last 24h tokens'),
      value: trendQuery.isError ? '-' : formatNumber(recentUsage.tokens),
      description: t('Tokens consumed in the last 24 hours'),
      icon: Flame,
      tone: 'accent-1' as const,
      sparkline: sparklines.tokens,
    },
    {
      key: 'totalTokens',
      title: t('Total tokens'),
      value: summaryQuery.isError
        ? '-'
        : formatNumber(summaryQuery.data?.data.total_tokens),
      description: t('Total tokens consumed'),
      icon: Layers,
      tone: 'accent-2' as const,
      sparkline: sparklines.tokens,
    },
    {
      key: 'requests',
      title: t('Request Count'),
      value: formatNumber(requestCount),
      description: t('Total requests made'),
      icon: Activity,
      tone: 'accent-3' as const,
      sparkline: sparklines.requests,
    },
  ]

  return (
    <section className='bg-card overflow-hidden rounded-2xl border p-3 shadow-xs sm:p-5'>
      <header className='mb-2.5 flex flex-col gap-1 sm:mb-3'>
        <h3 className='text-sm font-semibold sm:text-base'>
          {t('Usage at a glance')}
        </h3>
        <p className='text-muted-foreground text-xs sm:text-sm'>
          {t('Monitor token usage and request volume')}
        </p>
      </header>
      <StaggerContainer className='grid grid-cols-3 gap-1.5 sm:gap-3'>
        {items.map((item) => (
          <StaggerItem
            key={item.key}
            className='bg-background/60 rounded-lg border px-2 py-1.5 sm:rounded-xl sm:p-3'
          >
            <StatCard
              title={item.title}
              value={item.value}
              description={item.description}
              icon={item.icon}
              tone={item.tone}
              sparkline={item.sparkline}
              sparklineVariant='line'
              loading={loading}
              compactMobile
            />
          </StaggerItem>
        ))}
      </StaggerContainer>
    </section>
  )
}
