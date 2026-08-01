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
export type ChannelMonitorSettings = {
  enabled: boolean
  interval_minutes: number
  concurrency: number
  timeout_seconds: number
  confirm_retries: number
  confirm_retry_delay_seconds: number
  failure_threshold: number
  exclude_patterns: string[]
}

export type ChannelMonitorTask = {
  task_id: string
  status: 'pending' | 'running' | 'succeeded' | 'failed'
  state?: { progress?: number }
  error?: string
}

export type ChannelMonitorTarget = {
  channel_id: number
  channel_name: string
  groups: string
  model: string
  status: 'success' | 'failure'
  health: 'healthy' | 'degraded' | 'down'
  state_changed: boolean
  attempts: number
  latency_ms: number
  http_status: number
  error: string
  created_at: number
  success_rate_24h: number
  samples_24h: number
}

export type ChannelMonitorHistory = Omit<
  ChannelMonitorTarget,
  'success_rate_24h' | 'samples_24h'
> & { id: number }

export type ChannelMonitorOverviewResponse = {
  success: boolean
  message: string
  data?: {
    settings: ChannelMonitorSettings
    targets: ChannelMonitorTarget[]
    task: ChannelMonitorTask | null
  }
}

export type ChannelMonitorHistoryResponse = {
  success: boolean
  message: string
  data?: ChannelMonitorHistory[]
}

export type ChannelMonitorConfigResponse = {
  success: boolean
  message: string
  data?: ChannelMonitorSettings
}

export type ChannelMonitorTriggerResponse = {
  success: boolean
  message: string
  data?: { created: boolean; task: ChannelMonitorTask }
}

export type ChannelMonitorTaskResponse = {
  success: boolean
  message: string
  data?: ChannelMonitorTask
}
