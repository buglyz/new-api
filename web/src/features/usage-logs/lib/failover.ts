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
import type { UsageLog } from '../data/schema'
import { parseLogOther } from './format'

export interface FailoverTrace {
  attemptChannelIds: number[]
  retryCount: number
  finalSuccessfulChannelId: number | null
  requestId: string | null
}

export interface RelatedLogSearch {
  page: number
  requestId: string
  startTime?: number
  endTime?: number
}

export interface ObservedFailover {
  log: UsageLog
  trace: FailoverTrace
}

const RELATED_LOG_WINDOW_SECONDS = 5 * 60

export function getFailoverTrace(log: UsageLog): FailoverTrace | null {
  const rawChannels = parseLogOther(log.other)?.admin_info?.use_channel
  if (!Array.isArray(rawChannels)) return null

  const attemptChannelIds = rawChannels
    .map((channelId) => Number(channelId))
    .filter((channelId) => Number.isInteger(channelId) && channelId > 0)
  if (attemptChannelIds.length <= 1) return null

  return {
    attemptChannelIds,
    retryCount: attemptChannelIds.length - 1,
    finalSuccessfulChannelId:
      log.type === 2 && log.channel > 0 ? log.channel : null,
    requestId: log.request_id || null,
  }
}

export function getRelatedLogSearch(log: UsageLog): RelatedLogSearch | null {
  const requestId = log.request_id.trim()
  if (!requestId) return null

  const search: RelatedLogSearch = { page: 1, requestId }
  if (Number.isFinite(log.created_at) && log.created_at > 0) {
    search.startTime = Math.max(
      1,
      (Math.floor(log.created_at) - RELATED_LOG_WINDOW_SECONDS) * 1000
    )
    search.endTime =
      (Math.ceil(log.created_at) + RELATED_LOG_WINDOW_SECONDS) * 1000
  }
  return search
}

export function getObservedFailovers(logs: UsageLog[]): ObservedFailover[] {
  const observed: ObservedFailover[] = []
  const seenRequests = new Set<string>()

  for (const log of logs) {
    const trace = getFailoverTrace(log)
    if (!trace) continue

    const requestKey = trace.requestId?.trim()
      ? `request:${trace.requestId.trim()}`
      : `log:${log.id}`
    if (seenRequests.has(requestKey)) continue

    seenRequests.add(requestKey)
    observed.push({ log, trace })
  }

  return observed
}
