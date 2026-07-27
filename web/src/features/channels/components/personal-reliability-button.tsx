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
import { ShieldCheck } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { useStatus } from '@/hooks/use-status'
import { isPersonalModeEnabled } from '@/lib/personal-mode'

import { useChannels } from './channels-provider'

export function PersonalReliabilityButton() {
  const { t } = useTranslation()
  const { status } = useStatus()
  const { setOpen } = useChannels()
  if (!isPersonalModeEnabled(status)) return null

  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <Button
            variant='outline'
            size='icon'
            onClick={() => setOpen('personal-reliability')}
            aria-label={t('Reliability Operations')}
          />
        }
      >
        <ShieldCheck />
      </TooltipTrigger>
      <TooltipContent>{t('Reliability Operations')}</TooltipContent>
    </Tooltip>
  )
}
