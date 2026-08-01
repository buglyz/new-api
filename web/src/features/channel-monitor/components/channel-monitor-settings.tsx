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
import { zodResolver } from '@hookform/resolvers/zod'
import { useEffect } from 'react'
import { useForm, type Resolver } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { z } from 'zod'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import { Textarea } from '@/components/ui/textarea'

import type { ChannelMonitorSettings } from '../types'

const schema = z.object({
  enabled: z.boolean(),
  interval_minutes: z.coerce.number().int().min(1).max(1440),
  concurrency: z.coerce.number().int().min(1).max(32),
  timeout_seconds: z.coerce.number().int().min(1).max(120),
  confirm_retries: z.coerce.number().int().min(0).max(3),
  confirm_retry_delay_seconds: z.coerce.number().int().min(0).max(60),
  failure_threshold: z.coerce.number().int().min(1).max(10),
  exclude_patterns: z.string(),
})

type FormValues = z.output<typeof schema>

type ChannelMonitorSettingsProps = {
  settings: ChannelMonitorSettings
  disabled: boolean
  onSave: (settings: ChannelMonitorSettings) => Promise<unknown>
}

type NumericFieldName = Exclude<
  keyof FormValues,
  'enabled' | 'exclude_patterns'
>

function toFormValues(settings: ChannelMonitorSettings): FormValues {
  return {
    ...settings,
    exclude_patterns: settings.exclude_patterns.join('\n'),
  }
}

export function ChannelMonitorSettingsForm(props: ChannelMonitorSettingsProps) {
  const { t } = useTranslation()
  const form = useForm<FormValues>({
    resolver: zodResolver(schema) as unknown as Resolver<FormValues>,
    defaultValues: toFormValues(props.settings),
  })

  useEffect(() => {
    if (!form.formState.isDirty) {
      form.reset(toFormValues(props.settings))
    }
  }, [form, form.formState.isDirty, props.settings])

  const onSubmit = async (values: FormValues) => {
    await props.onSave({
      ...values,
      exclude_patterns: values.exclude_patterns
        .split('\n')
        .map((pattern) => pattern.trim())
        .filter(Boolean),
    })
    form.reset(values)
  }

  return (
    <form
      className='grid gap-4 border-b px-4 py-4 sm:px-5'
      onSubmit={(event) => void form.handleSubmit(onSubmit)(event)}
    >
      <div className='flex items-center justify-between gap-4'>
        <label
          className='text-sm font-medium'
          htmlFor='channel-monitor-enabled'
        >
          {t('Enable channel monitoring')}
        </label>
        <Switch
          id='channel-monitor-enabled'
          checked={form.watch('enabled')}
          disabled={props.disabled}
          onCheckedChange={(enabled) =>
            form.setValue('enabled', enabled, { shouldDirty: true })
          }
        />
      </div>
      <div className='grid gap-3 sm:grid-cols-2 lg:grid-cols-3'>
        <NumericField
          form={form}
          name='interval_minutes'
          label={t('Interval (minutes)')}
          disabled={props.disabled}
        />
        <NumericField
          form={form}
          name='concurrency'
          label={t('Concurrency')}
          disabled={props.disabled}
        />
        <NumericField
          form={form}
          name='timeout_seconds'
          label={t('Timeout (seconds)')}
          disabled={props.disabled}
        />
        <NumericField
          form={form}
          name='confirm_retries'
          label={t('Failure retries')}
          disabled={props.disabled}
        />
        <NumericField
          form={form}
          name='confirm_retry_delay_seconds'
          label={t('Retry delay (seconds)')}
          disabled={props.disabled}
        />
        <NumericField
          form={form}
          name='failure_threshold'
          label={t('Failure threshold')}
          disabled={props.disabled}
        />
      </div>
      <div className='grid gap-1.5'>
        <label
          className='text-sm font-medium'
          htmlFor='channel-monitor-exclusions'
        >
          {t('Excluded model patterns')}
        </label>
        <Textarea
          id='channel-monitor-exclusions'
          disabled={props.disabled}
          aria-invalid={Boolean(form.formState.errors.exclude_patterns)}
          {...form.register('exclude_patterns')}
        />
        <FormError message={form.formState.errors.exclude_patterns?.message} />
      </div>
      <div className='flex justify-end'>
        <Button
          type='submit'
          disabled={props.disabled || !form.formState.isDirty}
        >
          {t('Save changes')}
        </Button>
      </div>
    </form>
  )
}

type NumericFieldProps = {
  form: ReturnType<typeof useForm<FormValues>>
  name: NumericFieldName
  label: string
  disabled: boolean
}

function NumericField(props: NumericFieldProps) {
  const error = props.form.formState.errors[props.name]?.message
  return (
    <div className='grid gap-1.5'>
      <label
        className='text-sm font-medium'
        htmlFor={`channel-monitor-${props.name}`}
      >
        {props.label}
      </label>
      <Input
        id={`channel-monitor-${props.name}`}
        type='number'
        min={0}
        disabled={props.disabled}
        aria-invalid={Boolean(error)}
        {...props.form.register(props.name)}
      />
      <FormError message={error} />
    </div>
  )
}

function FormError({ message }: { message?: string }) {
  if (!message) return null
  return <span className='text-destructive text-xs'>{message}</span>
}
