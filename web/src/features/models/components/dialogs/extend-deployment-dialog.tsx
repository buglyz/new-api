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
import { useQueryClient } from '@tanstack/react-query'
import { Loader2 } from 'lucide-react'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Dialog } from '@/components/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { extendDeployment } from '../../api'
import { deploymentsQueryKeys } from '../../lib'

function toInt(value: unknown, fallback: number) {
  const n = typeof value === 'number' ? value : Number(value)
  return Number.isFinite(n) ? Math.max(0, Math.round(n)) : fallback
}

export function ExtendDeploymentDialog({
  open,
  onOpenChange,
  deploymentId,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  deploymentId: string | number | null
}) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [hours, setHours] = useState(1)
  const [isSubmitting, setIsSubmitting] = useState(false)

  useEffect(() => {
    if (open) setHours(1)
  }, [open])

  const canSubmit = Boolean(deploymentId) && hours > 0 && !isSubmitting

  const onSubmit = async () => {
    if (!deploymentId) return
    const h = toInt(hours, 1)
    if (h <= 0) {
      toast.error(t('Please enter a valid duration'))
      return
    }
    setIsSubmitting(true)
    try {
      const res = await extendDeployment(deploymentId, h)
      if (res.success) {
        toast.success(t('Extended successfully'))
        queryClient.invalidateQueries({
          queryKey: deploymentsQueryKeys.lists(),
        })
        queryClient.invalidateQueries({ queryKey: ['deployment-details'] })
        onOpenChange(false)
        return
      }
      toast.error(res.message || t('Extend failed'))
    } catch (err: unknown) {
      toast.error(err instanceof Error ? err.message : t('Extend failed'))
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      title={t('Extend deployment')}
      contentClassName='sm:max-w-lg'
      footerClassName='mt-4'
      contentHeight='auto'
      bodyClassName='space-y-4'
      footer={
        <>
          <Button variant='outline' onClick={() => onOpenChange(false)}>
            {t('Cancel')}
          </Button>
          <Button onClick={() => void onSubmit()} disabled={!canSubmit}>
            {isSubmitting ? (
              <Loader2 className='mr-2 h-4 w-4 animate-spin' />
            ) : null}
            {t('Extend')}
          </Button>
        </>
      }
    >
      <div className='space-y-4'>
        <div className='text-muted-foreground text-sm'>
          {t('Deployment ID')}:{' '}
          <span className='font-mono'>{deploymentId}</span>
        </div>

        <div className='space-y-2'>
          <div className='text-sm font-medium'>{t('Duration (hours)')}</div>
          <Input
            type='number'
            min={1}
            value={hours}
            onChange={(e) => setHours(toInt(e.target.value, 1))}
          />
          <div className='text-muted-foreground text-xs'>
            {t('This will extend the deployment by the specified hours.')}
          </div>
        </div>
      </div>
    </Dialog>
  )
}
