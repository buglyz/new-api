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
import { api } from '@/lib/api'

import type {
  ChannelMonitorConfigResponse,
  ChannelMonitorOverviewResponse,
  ChannelMonitorSettings,
  ChannelMonitorTaskResponse,
  ChannelMonitorTriggerResponse,
} from './types'

export async function getChannelMonitorOverview(filter = 'all') {
  const response = await api.get<ChannelMonitorOverviewResponse>(
    '/api/channel-monitor/overview',
    { params: { filter } }
  )
  return response.data
}

export async function updateChannelMonitorConfig(
  settings: ChannelMonitorSettings
) {
  const response = await api.put<ChannelMonitorConfigResponse>(
    '/api/channel-monitor/config',
    settings
  )
  return response.data
}

export async function triggerChannelMonitor() {
  const response = await api.post<ChannelMonitorTriggerResponse>(
    '/api/channel-monitor/trigger'
  )
  return response.data
}

export async function getChannelMonitorTask(taskID: string) {
  const response = await api.get<ChannelMonitorTaskResponse>(
    `/api/system-task/${taskID}`
  )
  return response.data
}
