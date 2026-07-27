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
import { Link } from '@tanstack/react-router'
import { AlertTriangle, ArrowRight, GitBranch, KeyRound } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { StatusBadge } from '@/components/status-badge'
import { Button } from '@/components/ui/button'
import { channelsQueryKeys } from '@/features/channels/lib/channel-actions'
import { channelNeedsAttention } from '@/features/channels/lib/channel-attention'
import { fetchAllChannelPages } from '@/features/channels/lib/channel-pages'
import { apiKeyNeedsAttention } from '@/features/keys/lib/api-key-attention'
import { fetchAllApiKeyPages } from '@/features/keys/lib/api-key-pages'
import { getAllLogs } from '@/features/usage-logs/api'
import type { UsageLog } from '@/features/usage-logs/data/schema'
import {
  getObservedFailovers,
  getRelatedLogSearch,
} from '@/features/usage-logs/lib/failover'
import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'

const SUMMARY_STALE_TIME_MS = 5 * 60 * 1000

export function PersonalOperationsSummary() {
  const { t } = useTranslation()
  const isAdmin = useAuthStore(
    (state) => (state.auth.user?.role ?? 0) >= ROLE.ADMIN
  )
  const nowSeconds = Date.now() / 1000

  const channelsQuery = useQuery({
    queryKey: [...channelsQueryKeys.lists(), 'dashboard-personal-operations'],
    queryFn: () => fetchAllChannelPages({}, false),
    enabled: isAdmin,
    staleTime: SUMMARY_STALE_TIME_MS,
    refetchOnWindowFocus: false,
    retry: false,
  })
  const keysQuery = useQuery({
    queryKey: ['dashboard', 'personal-operations', 'keys'],
    queryFn: () => fetchAllApiKeyPages({}),
    enabled: isAdmin,
    staleTime: SUMMARY_STALE_TIME_MS,
    refetchOnWindowFocus: false,
    retry: false,
  })
  const logsQuery = useQuery({
    queryKey: ['dashboard', 'personal-operations', 'failovers'],
    queryFn: async () => {
      const result = await getAllLogs({
        p: 1,
        page_size: 50,
        start_timestamp: Math.floor(Date.now() / 1000) - 24 * 60 * 60,
      })
      if (!result.success) {
        throw new Error(result.message || 'Failed to load recent logs')
      }
      return (result.data?.items ?? []) as UsageLog[]
    },
    enabled: isAdmin,
    staleTime: SUMMARY_STALE_TIME_MS,
    refetchOnWindowFocus: false,
    retry: false,
  })

  const channelAttentionCount = (channelsQuery.data ?? []).filter((channel) =>
    channelNeedsAttention(channel, nowSeconds)
  ).length
  const keyAttentionCount = (keysQuery.data ?? []).filter((apiKey) =>
    apiKeyNeedsAttention(apiKey, nowSeconds)
  ).length
  const observedFailovers = getObservedFailovers(logsQuery.data ?? [])
  const recentFailovers = observedFailovers.slice(0, 3)

  const channelValue = queryValue(channelsQuery, channelAttentionCount, t)
  const keyValue = queryValue(keysQuery, keyAttentionCount, t)
  const failoverValue = queryValue(logsQuery, observedFailovers.length, t)

  if (!isAdmin) return null

  return (
    <section className='bg-card overflow-hidden rounded-lg border'>
      <header className='flex flex-wrap items-start justify-between gap-3 border-b px-4 py-3'>
        <div>
          <h2 className='text-sm font-semibold'>{t('Personal Operations')}</h2>
          <p className='text-muted-foreground mt-0.5 text-xs'>
            {t(
              'Channel signals use status and latest probes, not traffic success rate.'
            )}
          </p>
        </div>
        <StatusBadge
          label={t('5 minute cache')}
          variant='neutral'
          copyable={false}
        />
      </header>

      <div className='grid md:grid-cols-3'>
        <SummaryItem
          icon={<AlertTriangle />}
          title={t('Channels Needing Attention')}
          value={channelValue}
          description={t(
            'Automatic disablement, missing or stale probes, slow probes, and unavailable multi-keys.'
          )}
          to='/channels'
        />
        <SummaryItem
          icon={<KeyRound />}
          title={t('API Key Warnings')}
          value={keyValue}
          description={t(
            'Expiry, quota, inactivity, unlimited access, and model restriction warnings.'
          )}
          to='/keys'
        />
        <SummaryItem
          icon={<GitBranch />}
          title={t('Recent Observed Failovers')}
          value={failoverValue}
          description={t(
            'Observed in the latest 50 logs from the last 24 hours.'
          )}
          to='/usage-logs'
        >
          {recentFailovers.map(({ log, trace }) => (
            <Link
              key={`${log.request_id}-${log.id}`}
              to='/usage-logs/$section'
              params={{ section: 'common' }}
              search={getRelatedLogSearch(log) ?? { page: 1 }}
              className='text-muted-foreground hover:text-foreground block truncate font-mono text-[11px]'
            >
              {trace.attemptChannelIds.join(' → ')} · {log.request_id}
            </Link>
          ))}
        </SummaryItem>
      </div>
    </section>
  )
}

function queryValue(
  query: { isPending: boolean; isError: boolean },
  count: number,
  t: (key: string) => string
): string {
  if (query.isPending) return '—'
  return query.isError ? t('Unavailable') : String(count)
}

function SummaryItem(props: {
  icon: React.ReactNode
  title: string
  value: string
  description: string
  to: '/channels' | '/keys' | '/usage-logs'
  children?: React.ReactNode
}) {
  return (
    <div className='border-b p-4 last:border-b-0 md:border-r md:border-b-0 md:last:border-r-0'>
      <div className='flex items-start justify-between gap-3'>
        <div className='text-warning [&_svg]:size-4'>{props.icon}</div>
        <span className='text-2xl font-semibold tabular-nums'>
          {props.value}
        </span>
      </div>
      <h3 className='mt-2 text-sm font-medium'>{props.title}</h3>
      <p className='text-muted-foreground mt-1 text-xs leading-relaxed'>
        {props.description}
      </p>
      {props.children && <div className='mt-2 space-y-1'>{props.children}</div>}
      <Button
        variant='ghost'
        size='sm'
        className='mt-2 h-7 px-0'
        render={<Link to={props.to} />}
      >
        {props.title}
        <ArrowRight data-icon='inline-end' />
      </Button>
    </div>
  )
}
