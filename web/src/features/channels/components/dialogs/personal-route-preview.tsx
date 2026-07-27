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
import { Route } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { StatusBadge } from '@/components/status-badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

import { simulatePersonalRoute } from '../../personal-reliability-api'
import type { PersonalRoutePreviewResponse } from '../../personal-reliability-types'

export function PersonalRoutePreview() {
  const { t } = useTranslation()
  const [group, setGroup] = useState('default')
  const [model, setModel] = useState('')
  const [requestPath, setRequestPath] = useState('/v1/chat/completions')
  const [loading, setLoading] = useState(false)
  const [result, setResult] = useState<PersonalRoutePreviewResponse['data']>()

  const simulate = async () => {
    if (!group.trim() || !model.trim()) {
      toast.error(t('Group and model are required'))
      return
    }
    setLoading(true)
    try {
      const response = await simulatePersonalRoute({
        group: group.trim(),
        model: model.trim(),
        request_path: requestPath.trim(),
      })
      if (!response.success) {
        toast.error(response.message || t('Route preview failed'))
        return
      }
      setResult(response.data)
    } catch {
      toast.error(t('Route preview failed'))
    } finally {
      setLoading(false)
    }
  }

  return (
    <section className='space-y-3 border-t pt-4'>
      <h3 className='flex items-center gap-2 text-sm font-medium'>
        <Route className='size-4' />
        {t('Route Preview')}
      </h3>
      <div className='grid gap-3 sm:grid-cols-2'>
        <div className='space-y-1.5'>
          <Label htmlFor='route-preview-group'>{t('Group')}</Label>
          <Input
            id='route-preview-group'
            value={group}
            onChange={(event) => setGroup(event.target.value)}
          />
        </div>
        <div className='space-y-1.5'>
          <Label htmlFor='route-preview-model'>{t('Model')}</Label>
          <Input
            id='route-preview-model'
            value={model}
            onChange={(event) => setModel(event.target.value)}
          />
        </div>
      </div>
      <div className='flex gap-2'>
        <Input
          aria-label={t('Request Path')}
          value={requestPath}
          onChange={(event) => setRequestPath(event.target.value)}
        />
        <Button onClick={simulate} disabled={loading}>
          {t('Preview')}
        </Button>
      </div>
      {result && (
        <div className='divide-y rounded-md border text-xs'>
          {result.candidates.length === 0 ? (
            <p className='text-muted-foreground p-3'>
              {t('No route candidates')}
            </p>
          ) : (
            result.candidates.map((candidate) => (
              <div
                key={candidate.channel_id}
                className='flex flex-wrap items-center gap-2 px-3 py-2'
              >
                <span className='font-mono'>#{candidate.channel_id}</span>
                <span className='min-w-0 flex-1 truncate'>
                  {candidate.channel_name}
                </span>
                <span className='text-muted-foreground'>
                  P{candidate.priority} · W{candidate.weight}
                </span>
                <StatusBadge
                  label={t(
                    candidate.eligible ? 'Route eligible' : 'Cooling down'
                  )}
                  variant={candidate.eligible ? 'green' : 'warning'}
                  copyable={false}
                  size='sm'
                />
              </div>
            ))
          )}
        </div>
      )}
    </section>
  )
}
