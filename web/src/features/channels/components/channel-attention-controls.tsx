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
import { Loader2, RefreshCw } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'

export function ChannelAttentionControls(props: {
  attentionOnly: boolean
  attentionCount: number
  isFetching: boolean
  updatedAt: number
  onModeChange: (attentionOnly: boolean) => void
  onRefresh: () => void
}) {
  const { t } = useTranslation()
  const activeValue = props.attentionOnly ? 'attention' : 'all'
  const updatedLabel = props.updatedAt
    ? new Date(props.updatedAt).toLocaleTimeString()
    : t('Not refreshed yet')

  return (
    <div className='flex items-center gap-1.5'>
      <ToggleGroup
        value={[activeValue]}
        onValueChange={(values) => {
          const next = values.find((value) => value !== activeValue)
          if (next) props.onModeChange(next === 'attention')
        }}
        variant='outline'
        size='sm'
        aria-label={t('Channel view')}
      >
        <ToggleGroupItem value='all'>{t('All')}</ToggleGroupItem>
        <ToggleGroupItem value='attention'>
          {t('Needs Attention')} ({props.attentionCount})
        </ToggleGroupItem>
      </ToggleGroup>
      <Tooltip>
        <TooltipTrigger
          render={
            <Button
              variant='ghost'
              size='icon'
              className='size-7'
              onClick={props.onRefresh}
              disabled={props.isFetching}
              aria-label={t('Refresh channel attention summary')}
            />
          }
        >
          {props.isFetching ? (
            <Loader2 className='animate-spin' />
          ) : (
            <RefreshCw />
          )}
        </TooltipTrigger>
        <TooltipContent>
          {t('Last refreshed')}: {updatedLabel}
        </TooltipContent>
      </Tooltip>
    </div>
  )
}
