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
import { CHANNEL_STATUS, MULTI_KEY_STATUS } from '../constants'
import type { Channel } from '../types'

export const CHANNEL_PROBE_STALE_SECONDS = 24 * 60 * 60
export const CHANNEL_SLOW_PROBE_MS = 5_000

export type ChannelAttentionReason =
  | 'auto_disabled'
  | 'all_keys_unavailable'
  | 'untested'
  | 'stale_probe'
  | 'slow_probe'

export const CHANNEL_ATTENTION_LABELS: Record<ChannelAttentionReason, string> =
  {
    auto_disabled: 'Automatically disabled',
    all_keys_unavailable: 'All multi-key credentials unavailable',
    untested: 'Never probed',
    stale_probe: 'Probe older than 24 hours',
    slow_probe: 'Probe response over 5 seconds',
  }

function areAllMultiKeysUnavailable(channel: Channel): boolean {
  const info = channel.channel_info
  if (!info?.is_multi_key || info.multi_key_size <= 0) return false
  const unavailableCount = Object.values(
    info.multi_key_status_list ?? {}
  ).filter((status) => status !== MULTI_KEY_STATUS.ENABLED).length
  return unavailableCount >= info.multi_key_size
}

export function getChannelAttentionReasons(
  channel: Channel,
  nowSeconds: number = Date.now() / 1000
): ChannelAttentionReason[] {
  const reasons: ChannelAttentionReason[] = []

  if (channel.status === CHANNEL_STATUS.AUTO_DISABLED) {
    reasons.push('auto_disabled')
  }
  if (areAllMultiKeysUnavailable(channel)) {
    reasons.push('all_keys_unavailable')
  }
  if (!channel.test_time || channel.test_time <= 0) {
    reasons.push('untested')
  } else if (nowSeconds - channel.test_time > CHANNEL_PROBE_STALE_SECONDS) {
    reasons.push('stale_probe')
  }
  if (channel.response_time > CHANNEL_SLOW_PROBE_MS) {
    reasons.push('slow_probe')
  }

  return reasons
}

export function channelNeedsAttention(
  channel: Channel,
  nowSeconds?: number
): boolean {
  return getChannelAttentionReasons(channel, nowSeconds).length > 0
}
