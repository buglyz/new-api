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
import { API_KEY_STATUS } from '../constants'
import type { ApiKey } from '../types'

export const API_KEY_EXPIRING_SOON_SECONDS = 7 * 24 * 60 * 60
export const API_KEY_UNUSED_SECONDS = 90 * 24 * 60 * 60

export type ApiKeyAttentionReason =
  | 'expiring_soon'
  | 'exhausted'
  | 'long_unused'
  | 'unbounded'
  | 'no_model_limits'

export const API_KEY_ATTENTION_LABELS: Record<ApiKeyAttentionReason, string> = {
  expiring_soon: 'Expires within 7 days',
  exhausted: 'Quota exhausted',
  long_unused: 'Unused for over 90 days',
  unbounded: 'No expiry and unlimited quota',
  no_model_limits: 'No model restrictions',
}

export function getApiKeyAttentionReasons(
  apiKey: ApiKey,
  nowSeconds: number = Date.now() / 1000
): ApiKeyAttentionReason[] {
  const reasons: ApiKeyAttentionReason[] = []
  const expiresIn = apiKey.expired_time - nowSeconds
  if (
    apiKey.expired_time > 0 &&
    expiresIn > 0 &&
    expiresIn <= API_KEY_EXPIRING_SOON_SECONDS
  ) {
    reasons.push('expiring_soon')
  }
  if (
    apiKey.status === API_KEY_STATUS.EXHAUSTED ||
    (!apiKey.unlimited_quota && apiKey.remain_quota <= 0)
  ) {
    reasons.push('exhausted')
  }

  const lastActivity = apiKey.accessed_time || apiKey.created_time
  if (lastActivity > 0 && nowSeconds - lastActivity > API_KEY_UNUSED_SECONDS) {
    reasons.push('long_unused')
  }
  if (apiKey.expired_time === -1 && apiKey.unlimited_quota) {
    reasons.push('unbounded')
  }
  if (!apiKey.model_limits_enabled) {
    reasons.push('no_model_limits')
  }

  return reasons
}

export function apiKeyNeedsAttention(
  apiKey: ApiKey,
  nowSeconds?: number
): boolean {
  return getApiKeyAttentionReasons(apiKey, nowSeconds).length > 0
}
