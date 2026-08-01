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
import { Settings2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'

import type { ChannelMonitorChannel, ChannelMonitorSettings } from '../types'
import { ChannelMonitorSettingsForm } from './channel-monitor-settings'

type ChannelMonitorSettingsDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  settings: ChannelMonitorSettings
  channels: ChannelMonitorChannel[]
  disabled: boolean
  onSave: (settings: ChannelMonitorSettings) => Promise<unknown>
}

export function ChannelMonitorSettingsDialog(
  props: ChannelMonitorSettingsDialogProps
) {
  const { t } = useTranslation()
  return (
    <Dialog open={props.open} onOpenChange={props.onOpenChange}>
      <Tooltip>
        <TooltipTrigger
          render={
            <Button
              type='button'
              variant='outline'
              size='icon'
              onClick={() => props.onOpenChange(true)}
              aria-label={t('Monitor settings')}
            />
          }
        >
          <Settings2 className='size-4' aria-hidden='true' />
        </TooltipTrigger>
        <TooltipContent>{t('Monitor settings')}</TooltipContent>
      </Tooltip>
      <DialogContent className='max-h-[min(90vh,48rem)] overflow-y-auto sm:max-w-2xl'>
        <DialogHeader>
          <DialogTitle>{t('Monitor settings')}</DialogTitle>
          <DialogDescription>
            {t('Configure probe cadence, retries, exclusions, and thresholds.')}
          </DialogDescription>
        </DialogHeader>
        <ChannelMonitorSettingsForm
          settings={props.settings}
          channels={props.channels}
          disabled={props.disabled}
          onSave={props.onSave}
        />
      </DialogContent>
    </Dialog>
  )
}
