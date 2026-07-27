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

import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group'

export function ApiKeyAttentionControls(props: {
  attentionOnly: boolean
  attentionCount: number
  onModeChange: (attentionOnly: boolean) => void
}) {
  const { t } = useTranslation()
  const activeValue = props.attentionOnly ? 'attention' : 'all'

  return (
    <ToggleGroup
      value={[activeValue]}
      onValueChange={(values) => {
        const next = values.find((value) => value !== activeValue)
        if (next) props.onModeChange(next === 'attention')
      }}
      variant='outline'
      size='sm'
      aria-label={t('API key view')}
    >
      <ToggleGroupItem value='all'>{t('All')}</ToggleGroupItem>
      <ToggleGroupItem value='attention'>
        {t('Needs Attention')} ({props.attentionCount})
      </ToggleGroupItem>
    </ToggleGroup>
  )
}
