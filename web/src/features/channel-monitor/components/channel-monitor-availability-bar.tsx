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
import { useTranslation } from 'react-i18next'

import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import {
  getSuccessRateDotClass,
  getSuccessRateTextClass,
} from '@/features/performance-metrics/lib/format'
import { formatTimestampToDate } from '@/lib/format'
import { cn } from '@/lib/utils'

import type { ChannelMonitorAvailabilityPoint } from '../types'

type ChannelMonitorAvailabilityBarProps = {
  points: ChannelMonitorAvailabilityPoint[]
  overall: number | null
  showOverall?: boolean
  className?: string
}

function formatRate(rate: number | null) {
  return rate === null ? '-' : `${(rate * 100).toFixed(2)}%`
}

export function ChannelMonitorAvailabilityBar(
  props: ChannelMonitorAvailabilityBarProps
) {
  const { t } = useTranslation()
  const showOverall = props.showOverall ?? true

  if (props.points.length === 0) {
    return <span className='text-muted-foreground text-xs'>{t('No data')}</span>
  }

  return (
    <div className={cn('flex min-w-0 items-center gap-2', props.className)}>
      <div
        className='flex h-4 min-w-0 flex-1 gap-0.5'
        role='img'
        aria-label={`${t('Availability (last 24h)')}: ${formatRate(props.overall)}`}
      >
        {props.points.map((point) => {
          const rate = point.success_rate
          const color =
            rate === null
              ? 'bg-muted-foreground/20'
              : getSuccessRateDotClass(rate * 100)
          return (
            <Tooltip key={point.start_at}>
              <TooltipTrigger
                render={
                  <span className='min-w-0 flex-1 cursor-help rounded-sm outline-none' />
                }
              >
                <span
                  className={cn(
                    'block h-full min-h-2 rounded-sm transition-opacity hover:opacity-80',
                    color
                  )}
                />
              </TooltipTrigger>
              <TooltipContent side='top' className='font-mono text-xs'>
                <div className='font-medium'>
                  {formatTimestampToDate(point.start_at)}
                </div>
                {rate === null ? (
                  <div>{t('No data')}</div>
                ) : (
                  <>
                    <div>
                      {t('Success rate')}: {formatRate(rate)}
                    </div>
                    <div className='text-muted-foreground'>
                      {point.succeeded}/{point.samples}
                    </div>
                  </>
                )}
              </TooltipContent>
            </Tooltip>
          )
        })}
      </div>
      {showOverall && (
        <span
          className={cn(
            'shrink-0 font-mono text-xs font-semibold tabular-nums',
            props.overall === null
              ? 'text-muted-foreground'
              : getSuccessRateTextClass(props.overall * 100)
          )}
        >
          {formatRate(props.overall)}
        </span>
      )}
    </div>
  )
}
