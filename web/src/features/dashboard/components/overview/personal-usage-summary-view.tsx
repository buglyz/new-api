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
import { Activity, Flame, Layers } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { StaggerContainer, StaggerItem } from '@/components/page-transition'
import { formatTokenCountInMillions } from '@/features/dashboard/lib/personal-usage-summary'
import { formatNumber } from '@/lib/format'

import { StatCard } from '../ui/stat-card'

interface PersonalUsageSummaryViewProps {
  last24hTokens?: string
  totalTokens?: string
  requestCount: number
  tokensTracked: boolean
  summaryLoading: boolean
  summaryError: boolean
  tokenSparkline: number[]
  requestSparkline: number[]
}

export function PersonalUsageSummaryView(props: PersonalUsageSummaryViewProps) {
  const { t } = useTranslation()
  const trackingDescription = props.tokensTracked
    ? t('Total tokens consumed')
    : t('Usage tracking is disabled; totals are historical')
  const items = [
    {
      key: 'recentTokens',
      title: t('Last 24h tokens'),
      value: formatTokenCountInMillions(props.last24hTokens),
      description: t('Tokens consumed in the last 24 hours'),
      icon: Flame,
      tone: 'accent-1' as const,
      sparkline: props.tokenSparkline,
      loading: props.summaryLoading,
      error: props.summaryError,
    },
    {
      key: 'totalTokens',
      title: t('Total tokens'),
      value: formatTokenCountInMillions(props.totalTokens),
      description: trackingDescription,
      icon: Layers,
      tone: 'accent-2' as const,
      sparkline: props.tokenSparkline,
      loading: props.summaryLoading,
      error: props.summaryError,
    },
    {
      key: 'requests',
      title: t('Request Count'),
      value: formatNumber(props.requestCount),
      description: t('Total requests made'),
      icon: Activity,
      tone: 'accent-3' as const,
      sparkline: props.requestSparkline,
      loading: false,
      error: false,
    },
  ]

  return (
    <section
      className='bg-card overflow-hidden rounded-2xl border p-3 shadow-xs sm:p-5'
      data-personal-usage-summary='true'
    >
      <header className='mb-2.5 flex flex-col gap-1 sm:mb-3'>
        <h3 className='text-sm font-semibold sm:text-base'>
          {t('Usage at a glance')}
        </h3>
        <p className='text-muted-foreground text-xs sm:text-sm'>
          {t('Monitor token usage and request volume')}
        </p>
      </header>
      <StaggerContainer className='grid auto-rows-fr grid-cols-1 gap-2 sm:grid-cols-3 sm:gap-3'>
        {items.map((item) => (
          <StaggerItem
            key={item.key}
            className='bg-background/60 min-w-0 rounded-lg border px-3 py-2.5 sm:h-full sm:rounded-xl sm:p-3'
          >
            <StatCard
              title={item.title}
              value={item.value}
              description={item.description}
              icon={item.icon}
              tone={item.tone}
              sparkline={item.sparkline}
              sparklineVariant='line'
              loading={item.loading}
              error={item.error}
            />
          </StaggerItem>
        ))}
      </StaggerContainer>
    </section>
  )
}
