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
export interface PersonalCircuit {
  channel_id: number
  channel_name?: string
  model: string
  scope: 'channel' | 'model'
  status: 'open' | 'half_open'
  consecutive_failures: number
  opened_at: number
  retry_at: number
  half_open_until?: number
  last_outcome: string
  last_status_code?: number
  last_error_code?: string
}

export interface PersonalCircuitTransition {
  channel_id: number
  channel_name?: string
  model: string
  from: 'closed' | 'open' | 'half_open'
  to: 'closed' | 'open' | 'half_open'
  at: number
  outcome?: string
  status_code?: number
  error_code?: string
  retry_at?: number
}

export interface PersonalReliabilityResponse {
  success: boolean
  message?: string
  data?: {
    circuits: PersonalCircuit[]
    transitions: PersonalCircuitTransition[]
    policy: {
      base_backoff_seconds: number
      max_backoff_seconds: number
      model_backoff_seconds: number
      auth_backoff_seconds: number
      channel_backoff_seconds: number
      half_open_lease_seconds: number
      volatile: boolean
    }
  }
}

export interface PersonalRoutePreviewResponse {
  success: boolean
  message?: string
  data?: {
    group: string
    model: string
    request_path: string
    strategy: 'priority_then_weighted_random'
    highest_available_priority: number | null
    candidates: Array<{
      channel_id: number
      channel_name: string
      priority: number
      weight: number
      circuit_status: 'closed' | 'open' | 'half_open'
      retry_at?: number
      eligible: boolean
    }>
  }
}

export interface PersonalReliabilityTaskResponse {
  success: boolean
  message?: string
  data?: {
    task_id?: string
    status?: string
    reset_count?: number
  }
}
