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
  API_KEY_ATTENTION_LABELS,
  getApiKeyAttentionReasons,
} from '../lib/api-key-attention'
import type { ApiKey } from '../types'

export function ApiKeyAttentionBadges(props: {
  apiKey: ApiKey
  nowSeconds: number
}) {
  const { t } = useTranslation()
  const reasons = getApiKeyAttentionReasons(props.apiKey, props.nowSeconds)
  if (reasons.length === 0) return null

  return (
    <Tooltip>
      <TooltipTrigger render={<span className='inline-flex' />}>
        <StatusBadge
          label={
            reasons.length === 1
              ? t(API_KEY_ATTENTION_LABELS[reasons[0]])
              : t('{{count}} warning(s)', { count: reasons.length })
          }
          variant='warning'
          copyable={false}
        />
      </TooltipTrigger>
      <TooltipContent side='top' className='max-w-xs'>
        <ul className='space-y-1 text-xs'>
          {reasons.map((reason) => (
            <li key={reason}>{t(API_KEY_ATTENTION_LABELS[reason])}</li>
          ))}
        </ul>
      </TooltipContent>
    </Tooltip>
  )
}
