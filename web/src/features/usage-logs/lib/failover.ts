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
  attempts: FailoverAttempt[]
  retryCount: number
  finalSuccessfulChannelId: number | null
  requestId: string | null
  result: string | null
}

export interface FailoverAttempt {
  index: number
  channelId: number
  outcome: string | null
  statusCode: number | null
  errorCode: string | null
  durationMs: number | null
  retried: boolean
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

const ATTEMPT_OUTCOME_LABELS: Record<string, string> = {
  success: 'Success',
  transport_error: 'Transport error',
  rate_limited: 'Rate limited',
  upstream_5xx: 'Upstream server error',
  auth_error: 'Authentication error',
  model_unavailable: 'Model unavailable',
  channel_unavailable: 'Channel unavailable',
  client_error: 'Client error',
  local_error: 'Local error',
  upstream_error: 'Upstream error',
}

export function getAttemptOutcomeLabelKey(outcome: string): string {
  return ATTEMPT_OUTCOME_LABELS[outcome] ?? outcome
}

export function getFailoverTrace(log: UsageLog): FailoverTrace | null {
  const adminInfo = parseLogOther(log.other)?.admin_info
  const structured = adminInfo?.relay_attempts
  const structuredAttempts = Array.isArray(structured?.attempts)
    ? structured.attempts.flatMap((attempt, attemptIndex) => {
        const channelId = Number(attempt.channel_id)
        if (!Number.isInteger(channelId) || channelId <= 0) return []
        return [
          {
            index: finiteNumber(attempt.index) ?? attemptIndex,
            channelId,
            outcome:
              typeof attempt.outcome === 'string' ? attempt.outcome : null,
            statusCode: finiteNumber(attempt.status_code),
            errorCode:
              typeof attempt.error_code === 'string'
                ? attempt.error_code
                : null,
            durationMs: finiteNumber(attempt.duration_ms),
            retried: attempt.retried === true,
          },
        ]
      })
    : []
  if (structuredAttempts.length > 1) {
    const finalChannelId = Number(structured?.final_channel_id)
    return {
      attemptChannelIds: structuredAttempts.map((attempt) => attempt.channelId),
      attempts: structuredAttempts,
      retryCount: Math.max(
        Number.isInteger(structured?.retry_count)
          ? Number(structured?.retry_count)
          : structuredAttempts.length - 1,
        0
      ),
      finalSuccessfulChannelId:
        structured?.result === 'success' &&
        Number.isInteger(finalChannelId) &&
        finalChannelId > 0
          ? finalChannelId
          : null,
      requestId: structured?.request_id || log.request_id || null,
      result: structured?.result || null,
    }
  }

  const rawChannels = adminInfo?.use_channel
  if (!Array.isArray(rawChannels)) return null

  const attemptChannelIds = rawChannels
    .map((channelId) => Number(channelId))
    .filter((channelId) => Number.isInteger(channelId) && channelId > 0)
  if (attemptChannelIds.length <= 1) return null

  return {
    attemptChannelIds,
    attempts: attemptChannelIds.map((channelId, index) => ({
      index,
      channelId,
      outcome: null,
      statusCode: null,
      errorCode: null,
      durationMs: null,
      retried: index < attemptChannelIds.length - 1,
    })),
    retryCount: attemptChannelIds.length - 1,
    finalSuccessfulChannelId:
      log.type === 2 && log.channel > 0 ? log.channel : null,
    requestId: log.request_id || null,
    result: log.type === 2 ? 'success' : null,
  }
}

function finiteNumber(value: unknown): number | null {
  const number = Number(value)
  return Number.isFinite(number) && number >= 0 ? number : null
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
