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

import { StatusBadge } from '@/components/status-badge'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'

import {
  CHANNEL_ATTENTION_LABELS,
  getChannelAttentionReasons,
} from '../lib/channel-attention'
import type { Channel } from '../types'

export function ChannelAttentionBadges(props: {
  channel: Channel
  nowSeconds: number
}) {
  const { t } = useTranslation()
  const reasons = getChannelAttentionReasons(props.channel, props.nowSeconds)
  if (reasons.length === 0) return null

  const firstLabel = t(CHANNEL_ATTENTION_LABELS[reasons[0]])
  const extraCount = reasons.length - 1

  return (
    <Tooltip>
      <TooltipTrigger render={<span className='inline-flex shrink-0' />}>
        <StatusBadge
          label={extraCount > 0 ? `${firstLabel} +${extraCount}` : firstLabel}
          variant='warning'
          copyable={false}
          className='max-w-44'
        />
      </TooltipTrigger>
      <TooltipContent side='top' className='max-w-xs'>
        <ul className='space-y-1 text-xs'>
          {reasons.map((reason) => (
            <li key={reason}>{t(CHANNEL_ATTENTION_LABELS[reason])}</li>
          ))}
        </ul>
      </TooltipContent>
    </Tooltip>
  )
}
