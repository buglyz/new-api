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
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { Activity, RefreshCw, RotateCcw, TestTube } from 'lucide-react'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Dialog } from '@/components/dialog'
import { StatusBadge } from '@/components/status-badge'
import { Button } from '@/components/ui/button'

import { channelsQueryKeys } from '../../lib/channel-actions'
import { channelNeedsAttention } from '../../lib/channel-attention'
import { fetchAllChannelPages } from '../../lib/channel-pages'
import {
  canRunReliabilityBatch,
  getReliabilityRecoveryChannelIDs,
} from '../../lib/personal-reliability'
import {
  getPersonalReliability,
  resetPersonalReliabilityCircuits,
  runPersonalReliabilityTask,
} from '../../personal-reliability-api'
import { PersonalRoutePreview } from './personal-route-preview'

export function PersonalReliabilityDialog(props: {
  open: boolean
  onOpenChange: (open: boolean) => void
}) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [action, setAction] = useState<string | null>(null)
  const reliability = useQuery({
    queryKey: ['channel-reliability'],
    queryFn: getPersonalReliability,
    enabled: props.open,
    staleTime: 60_000,
    refetchInterval: props.open ? 60_000 : false,
  })
  const channels = useQuery({
    queryKey: [...channelsQueryKeys.lists(), 'reliability-tools'],
    queryFn: () => fetchAllChannelPages({}, false),
    enabled: props.open,
    staleTime: 5 * 60_000,
  })
  const attentionIDs = useMemo(
    () =>
      (channels.data ?? [])
        .filter((channel) => channelNeedsAttention(channel, Date.now() / 1000))
        .map((channel) => channel.id),
    [channels.data]
  )
  const circuits = reliability.data?.data?.circuits ?? []
  const circuitIDs = [...new Set(circuits.map((circuit) => circuit.channel_id))]
  const recoveryIDs = getReliabilityRecoveryChannelIDs(attentionIDs, circuitIDs)

  const runTask = async (kind: 'probe' | 'recover') => {
    const ids = kind === 'probe' ? attentionIDs : recoveryIDs
    if (!canRunReliabilityBatch(ids)) return
    setAction(kind)
    try {
      const response = await runPersonalReliabilityTask(kind, ids)
      if (!response.success) {
        toast.error(response.message || t('Operation failed'))
        return
      }
      toast.success(t('Channel test task queued'))
    } catch {
      toast.error(t('Operation failed'))
    } finally {
      setAction(null)
    }
  }

  const resetCircuits = async () => {
    if (!canRunReliabilityBatch(circuitIDs)) return
    setAction('reset')
    try {
      const response = await resetPersonalReliabilityCircuits(circuitIDs)
      if (!response.success) {
        toast.error(response.message || t('Operation failed'))
        return
      }
      await queryClient.invalidateQueries({ queryKey: ['channel-reliability'] })
      toast.success(t('Temporary circuits reset'))
    } catch {
      toast.error(t('Operation failed'))
    } finally {
      setAction(null)
    }
  }

  return (
    <Dialog
      open={props.open}
      onOpenChange={props.onOpenChange}
      title={t('Reliability Operations')}
      description={t('Personal channel and model reliability state')}
      contentClassName='sm:max-w-3xl'
      contentHeight='min(78dvh, 760px)'
      bodyClassName='space-y-4 pr-3'
    >
      <div className='flex flex-wrap gap-2'>
        <Button
          variant='outline'
          onClick={() => runTask('probe')}
          disabled={!canRunReliabilityBatch(attentionIDs) || action != null}
        >
          <TestTube data-icon='inline-start' />
          {t('Probe attention channels')} ({attentionIDs.length})
        </Button>
        <Button
          variant='outline'
          onClick={() => runTask('recover')}
          disabled={!canRunReliabilityBatch(recoveryIDs) || action != null}
        >
          <RefreshCw data-icon='inline-start' />
          {t('Probe and recover')}
        </Button>
        <Button
          variant='ghost'
          onClick={resetCircuits}
          disabled={circuitIDs.length === 0 || action != null}
        >
          <RotateCcw data-icon='inline-start' />
          {t('Reset temporary circuits')}
        </Button>
      </div>

      <section className='space-y-2'>
        <div className='flex items-center justify-between gap-2'>
          <h3 className='flex items-center gap-2 text-sm font-medium'>
            <Activity className='size-4' />
            {t('Temporary Circuits')}
          </h3>
          <StatusBadge
            label={t('Process-local state')}
            variant='neutral'
            copyable={false}
            size='sm'
          />
        </div>
        <div className='divide-y rounded-md border text-xs'>
          {circuits.length === 0 ? (
            <p className='text-muted-foreground p-3'>
              {t('No active temporary circuits')}
            </p>
          ) : (
            circuits.map((circuit) => (
              <div
                key={`${circuit.channel_id}-${circuit.model}`}
                className='flex flex-wrap items-center gap-2 px-3 py-2'
              >
                <span className='font-mono'>#{circuit.channel_id}</span>
                <span className='min-w-0 flex-1 truncate'>
                  {circuit.scope === 'channel'
                    ? t('All models')
                    : circuit.model}
                </span>
                {circuit.last_status_code && (
                  <span>HTTP {circuit.last_status_code}</span>
                )}
                <StatusBadge
                  label={t(circuit.status === 'open' ? 'Open' : 'Half open')}
                  variant='warning'
                  copyable={false}
                  size='sm'
                />
              </div>
            ))
          )}
        </div>
      </section>

      <PersonalRoutePreview />
    </Dialog>
  )
}
