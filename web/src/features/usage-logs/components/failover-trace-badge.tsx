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
import { useNavigate } from '@tanstack/react-router'
import { GitBranch, Search } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { StatusBadge } from '@/components/status-badge'
import { Button } from '@/components/ui/button'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'

import type { UsageLog } from '../data/schema'
import {
  getAttemptOutcomeLabelKey,
  getFailoverTrace,
  getRelatedLogSearch,
} from '../lib/failover'

export function FailoverTraceBadge(props: { log: UsageLog }) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const trace = getFailoverTrace(props.log)
  if (!trace) return null

  const chain = trace.attemptChannelIds.join(' → ')
  const relatedLogSearch = getRelatedLogSearch(props.log)
  const openRelatedLogs = () => {
    if (!relatedLogSearch) return
    void navigate({
      to: '/usage-logs/$section',
      params: { section: 'common' },
      search: relatedLogSearch,
    })
  }

  return (
    <Popover>
      <PopoverTrigger
        render={
          <button
            type='button'
            className='focus-visible:ring-ring rounded-md focus-visible:ring-2 focus-visible:outline-none'
            aria-label={t('Retry Chain')}
            onClick={(event) => event.stopPropagation()}
          />
        }
      >
        <StatusBadge
          icon={GitBranch}
          label={t('{{count}} retry(s)', { count: trace.retryCount })}
          variant='warning'
          copyable={false}
        />
      </PopoverTrigger>
      <PopoverContent side='top' align='start' className='w-72 text-xs'>
        <div className='space-y-2'>
          <p className='font-medium'>{t('Failover Trace')}</p>
          <dl className='grid grid-cols-[auto_minmax(0,1fr)] gap-x-2 gap-y-1'>
            <dt className='text-muted-foreground'>{t('Retry Count')}</dt>
            <dd>{trace.retryCount}</dd>
            <dt className='text-muted-foreground'>{t('Attempted Channels')}</dt>
            <dd className='font-mono break-all'>{chain}</dd>
            {trace.finalSuccessfulChannelId != null && (
              <>
                <dt className='text-muted-foreground'>
                  {t('Final Successful Channel')}
                </dt>
                <dd className='font-mono'>#{trace.finalSuccessfulChannelId}</dd>
              </>
            )}
            {trace.requestId && (
              <>
                <dt className='text-muted-foreground'>{t('Request ID')}</dt>
                <dd className='font-mono break-all'>{trace.requestId}</dd>
              </>
            )}
          </dl>
          <div className='border-t pt-2'>
            <p className='text-muted-foreground mb-1'>
              {t('Attempt Outcomes')}
            </p>
            <ol className='space-y-1'>
              {trace.attempts.map((attempt) => (
                <li
                  key={`${attempt.channelId}-${attempt.index}`}
                  className='flex flex-wrap items-baseline gap-x-1 font-mono'
                >
                  <span>#{attempt.channelId}</span>
                  {attempt.outcome && (
                    <span>{t(getAttemptOutcomeLabelKey(attempt.outcome))}</span>
                  )}
                  {attempt.statusCode != null && (
                    <span>HTTP {attempt.statusCode}</span>
                  )}
                  {attempt.durationMs != null && (
                    <span>{attempt.durationMs} ms</span>
                  )}
                </li>
              ))}
            </ol>
          </div>
          {relatedLogSearch && (
            <Button
              variant='outline'
              size='sm'
              className='w-full'
              onClick={openRelatedLogs}
            >
              <Search data-icon='inline-start' />
              {t('View Related Logs')}
            </Button>
          )}
        </div>
      </PopoverContent>
    </Popover>
  )
}
