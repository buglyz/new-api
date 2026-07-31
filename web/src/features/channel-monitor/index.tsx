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
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Play, RefreshCw } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ErrorState } from '@/components/error-state'
import { SectionPageLayout } from '@/components/layout'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'

import {
  getChannelMonitorOverview,
  triggerChannelMonitor,
  updateChannelMonitorConfig,
} from './api'
import { ChannelMonitorSettingsForm } from './components/channel-monitor-settings'
import {
  ChannelMonitorHistoryPanel,
  ChannelMonitorTable,
} from './components/channel-monitor-table'
import type { ChannelMonitorSettings, ChannelMonitorTarget } from './types'

export function ChannelMonitor() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [filter, setFilter] = useState('all')
  const [selected, setSelected] = useState<ChannelMonitorTarget | null>(null)
  const overviewQuery = useQuery({
    queryKey: ['channel-monitor', filter],
    queryFn: () => getChannelMonitorOverview(filter),
    refetchInterval: (query) => (query.state.data?.data?.task ? 5000 : false),
    retry: false,
  })
  const saveMutation = useMutation({
    mutationFn: async (settings: ChannelMonitorSettings) => {
      const response = await updateChannelMonitorConfig(settings)
      if (!response.success) throw new Error(response.message)
      return response
    },
    onSuccess: async () => {
      toast.success(t('Setting updated successfully'))
      await queryClient.invalidateQueries({ queryKey: ['channel-monitor'] })
    },
    onError: (error: Error) => {
      toast.error(error.message || t('Save failed'))
    },
  })
  const triggerMutation = useMutation({
    mutationFn: async () => {
      const response = await triggerChannelMonitor()
      if (!response.success) throw new Error(response.message)
      return response
    },
    onSuccess: async () => {
      toast.success(t('Monitor run queued'))
      await queryClient.invalidateQueries({ queryKey: ['channel-monitor'] })
    },
    onError: (error: Error) => {
      toast.error(error.message || t('Operation failed'))
    },
  })

  if (overviewQuery.isPending) {
    return <ChannelMonitorLoading title={t('Channel Monitor')} />
  }
  if (!overviewQuery.data?.success || !overviewQuery.data.data) {
    return (
      <SectionPageLayout>
        <SectionPageLayout.Title>{t('Channel Monitor')}</SectionPageLayout.Title>
        <SectionPageLayout.Content>
          <ErrorState
            title={t('Failed to load channel monitor')}
            onRetry={() => void overviewQuery.refetch()}
          />
        </SectionPageLayout.Content>
      </SectionPageLayout>
    )
  }

  const { settings, targets, task } = overviewQuery.data.data
  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>{t('Channel Monitor')}</SectionPageLayout.Title>
      <SectionPageLayout.Actions>
        <Tooltip>
          <TooltipTrigger
            render={
              <Button
                type='button'
                variant='outline'
                size='icon'
                onClick={() => void overviewQuery.refetch()}
                aria-label={t('Refresh')}
              />
            }
          >
            <RefreshCw className='size-4' aria-hidden='true' />
          </TooltipTrigger>
          <TooltipContent>{t('Refresh')}</TooltipContent>
        </Tooltip>
        <Button
          type='button'
          onClick={() => triggerMutation.mutate()}
          disabled={!settings.enabled || triggerMutation.isPending || task !== null}
        >
          <Play data-icon='inline-start' aria-hidden='true' />
          {t('Run monitor')}
        </Button>
      </SectionPageLayout.Actions>
      <SectionPageLayout.Content>
        <div className='space-y-4'>
          <div className='border'>
            <ChannelMonitorSettingsForm
              settings={settings}
              disabled={saveMutation.isPending}
              onSave={(next) => saveMutation.mutateAsync(next)}
            />
            <div className='px-4 py-3 sm:px-5'>
              <ToggleGroup
                value={[filter]}
                onValueChange={(values) => {
                  const next = values.find((value) => value !== filter)
                  if (next === 'all' || next === 'unhealthy') setFilter(next)
                }}
                variant='outline'
                size='sm'
                aria-label={t('Channel Monitor')}
              >
                <ToggleGroupItem value='all'>{t('All')}</ToggleGroupItem>
                <ToggleGroupItem value='unhealthy'>
                  {t('Abnormal')}
                </ToggleGroupItem>
              </ToggleGroup>
            </div>
          </div>
          <ChannelMonitorTable
            targets={targets}
            selected={selected}
            onSelect={setSelected}
          />
          <ChannelMonitorHistoryPanel target={selected} />
        </div>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}

function ChannelMonitorLoading({ title }: { title: string }) {
  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>{title}</SectionPageLayout.Title>
      <SectionPageLayout.Content>
        <div className='space-y-4'>
          <Skeleton className='h-56 w-full' />
          <Skeleton className='h-72 w-full' />
        </div>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
