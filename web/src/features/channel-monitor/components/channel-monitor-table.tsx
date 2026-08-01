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
import { Button } from '@/components/ui/button'
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '@/components/ui/collapsible'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { formatTimestampToDate } from '@/lib/format'

import { groupChannelMonitorTargets } from '../lib/channel-monitor-groups'
import type { ChannelMonitorAvailability, ChannelMonitorTarget } from '../types'
import { ChannelMonitorAvailabilityBar } from './channel-monitor-availability-bar'

type ChannelMonitorTableProps = {
  targets: ChannelMonitorTarget[]
  availability: ChannelMonitorAvailability[]
  selected: ChannelMonitorTarget | null
  onSelect: (target: ChannelMonitorTarget) => void
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
            <div className='overflow-x-auto'>
              <Table className='min-w-[860px]'>
                <TableHeader>
                  <TableRow>
                    <TableHead>{t('Model')}</TableHead>
                    <TableHead>{t('Health')}</TableHead>
                    <TableHead>{t('Latency')}</TableHead>
                    <TableHead>{t('24h success rate')}</TableHead>
                    <TableHead>{t('Last checked')}</TableHead>
                    <TableHead>{t('Detail')}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {group.targets.map((target) => (
                    <TableRow key={`${target.channel_id}-${target.model}`}>
                      <TableCell className='max-w-56 truncate font-mono'>
                        {target.model}
                      </TableCell>
                      <TableCell>
                        <Badge variant={healthVariant(target.health)}>
                          {t(target.health)}
                        </Badge>
                      </TableCell>
                      <TableCell>{target.latency_ms}ms</TableCell>
                      <TableCell>
                        {formatSuccessRate(
                          target.samples_24h ? target.success_rate_24h : null
                        )}
                      </TableCell>
                      <TableCell>
                        {formatTimestampToDate(target.created_at)}
                      </TableCell>
                      <TableCell>
                        <Button
                          type='button'
                          size='sm'
                          variant='outline'
                          onClick={() => props.onSelect(target)}
                        >
                          {t('History')}
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          </CollapsibleContent>
        </Collapsible>
      ))}
    </div>
  )
}
