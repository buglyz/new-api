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
import { ChevronDown } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Badge } from '@/components/ui/badge'
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '@/components/ui/collapsible'
import { formatTimestampToDate } from '@/lib/format'

import { groupChannelMonitorTargets } from '../lib/channel-monitor-groups'
import type { ChannelMonitorAvailability, ChannelMonitorTarget } from '../types'
import { ChannelMonitorAvailabilityBar } from './channel-monitor-availability-bar'

type ChannelMonitorTableProps = {
  targets: ChannelMonitorTarget[]
  availability: ChannelMonitorAvailability[]
}

function healthVariant(health: ChannelMonitorTarget['health']) {
  if (health === 'down') return 'destructive'
  if (health === 'degraded') return 'warning'
  return 'secondary'
}

function formatSuccessRate(rate: number | null) {
  return rate === null ? '-' : `${Math.round(rate * 100)}%`
}

export function ChannelMonitorTable(props: ChannelMonitorTableProps) {
  const { t } = useTranslation()
  const groups = groupChannelMonitorTargets(props.targets)
  const availabilityByChannel = new Map(
    props.availability.map((item) => [item.channel_id, item])
  )
  const availabilityByTarget = new Map(
    props.availability.flatMap((item) =>
      item.models.map(
        (model) => [`${item.channel_id}#${model.model}`, model] as const
      )
    )
  )

  if (groups.length === 0) {
    return (
      <div className='border'>
        <p className='text-muted-foreground px-4 py-8 text-center text-sm'>
          {t('No monitor results')}
        </p>
      </div>
    )
  }

  return (
    <div className='grid gap-2'>
      {groups.map((group) => (
        <Collapsible key={group.channel_id} className='border'>
          <CollapsibleTrigger className='group hover:bg-muted/50 flex w-full items-center justify-between gap-4 px-4 py-3 text-left transition-colors sm:px-5'>
            <div className='min-w-0 flex-1'>
              <div className='truncate font-medium'>{group.channel_name}</div>
              <div className='text-muted-foreground mt-1 text-xs'>
                {t('{{count}} models', { count: group.targets.length })}
                <span className='mx-1.5'>·</span>
                {formatTimestampToDate(group.last_checked)}
              </div>
              <ChannelMonitorAvailabilityBar
                className='mt-2 max-w-md'
                points={
                  availabilityByChannel.get(group.channel_id)?.points ?? []
                }
                overall={group.success_rate_24h}
              />
            </div>
            <div className='flex shrink-0 items-center gap-3'>
              <div className='text-right'>
                <div className='text-sm font-semibold'>
                  {formatSuccessRate(group.success_rate_24h)}
                </div>
                <div className='text-muted-foreground text-xs'>
                  {t('24h success rate')}
                </div>
              </div>
              <Badge variant={healthVariant(group.health)}>
                {t(group.health)}
              </Badge>
              <ChevronDown className='text-muted-foreground size-4 shrink-0 transition-transform group-data-[panel-open]/collapsible-trigger:rotate-180' />
            </div>
          </CollapsibleTrigger>
          <CollapsibleContent className='border-t'>
            <div className='divide-y'>
              {group.targets.map((target) => {
                const modelAvailability = availabilityByTarget.get(
                  `${target.channel_id}#${target.model}`
                )
                return (
                  <div
                    key={`${target.channel_id}-${target.model}`}
                    className='px-4 py-3 sm:px-5'
                  >
                    <div className='mb-2 truncate font-mono text-sm'>
                      {target.model}
                    </div>
                    <ChannelMonitorAvailabilityBar
                      points={modelAvailability?.points ?? []}
                      overall={
                        target.samples_24h ? target.success_rate_24h : null
                      }
                      showOverall={false}
                      className='w-full max-w-xl'
                    />
                  </div>
                )
              })}
            </div>
          </CollapsibleContent>
        </Collapsible>
      ))}
    </div>
  )
}
