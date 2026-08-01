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
import { useTranslation } from 'react-i18next'

import { ErrorState } from '@/components/error-state'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'
import { formatTimestampToDate } from '@/lib/format'

import { getChannelMonitorHistory } from '../api'
import type { ChannelMonitorTarget } from '../types'

function healthVariant(health: ChannelMonitorTarget['health']) {
  if (health === 'down') return 'destructive'
  if (health === 'degraded') return 'warning'
  return 'secondary'
}

export function ChannelMonitorHistoryPanel({
  target,
}: {
  target: ChannelMonitorTarget | null
}) {
  const { t } = useTranslation()
  const channelID = target?.channel_id
  const modelName = target?.model
  const historyQuery = useQuery({
    queryKey: ['channel-monitor-history', channelID, modelName],
    queryFn: () => {
      if (channelID === undefined || !modelName) {
        return Promise.reject(new Error('monitor target is required'))
      }
      return getChannelMonitorHistory(channelID, modelName)
    },
    enabled: channelID !== undefined && Boolean(modelName),
    retry: false,
  })

  if (!target) return null
  if (historyQuery.isPending) {
    return (
      <section className='border px-4 py-4 sm:px-5'>
        <Skeleton className='h-5 w-64' />
        <div className='mt-3 grid gap-2'>
          <Skeleton className='h-6 w-full' />
          <Skeleton className='h-6 w-5/6' />
        </div>
      </section>
    )
  }
  if (historyQuery.isError || !historyQuery.data?.success) {
    return (
      <ErrorState
        title={t('Failed to load monitor history')}
        onRetry={() => void historyQuery.refetch()}
      />
    )
  }

  const history = historyQuery.data.data ?? []
  return (
    <section className='border px-4 py-4 sm:px-5'>
      <h2 className='text-sm font-semibold'>
        {t('Probe history')}: {target.channel_name} / {target.model}
      </h2>
      <div className='mt-3 grid gap-2'>
        {history.map((entry) => (
          <div
            key={entry.id}
            className='flex flex-wrap items-center gap-x-3 gap-y-1 text-sm'
          >
            <span className='text-muted-foreground'>
              {formatTimestampToDate(entry.created_at)}
            </span>
            <Badge variant={healthVariant(entry.health)}>
              {t(entry.health)}
            </Badge>
            <span>{entry.latency_ms}ms</span>
            {entry.error && (
              <span className='text-destructive break-all'>{entry.error}</span>
            )}
          </div>
        ))}
        {history.length === 0 && (
          <span className='text-muted-foreground text-sm'>
            {t('No monitor history')}
          </span>
        )}
      </div>
    </section>
  )
}
