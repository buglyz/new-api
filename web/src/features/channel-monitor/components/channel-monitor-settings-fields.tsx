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
import type { UseFormReturn } from 'react-hook-form'

import { Input } from '@/components/ui/input'

import type { FormValues } from './channel-monitor-settings'

type NumericFieldName = Exclude<
  keyof FormValues,
  'enabled' | 'exclude_patterns' | 'exclude_channel_ids'
>

export function NumericField(props: {
  form: UseFormReturn<FormValues>
  name: NumericFieldName
  label: string
  disabled: boolean
}) {
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

export function FormError({ message }: { message?: string }) {
  if (!message) return null
  return <span className='text-destructive text-xs'>{message}</span>
}
