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
import type { ChannelMonitorHealth, ChannelMonitorTarget } from '../types'

export type ChannelMonitorGroup = {
  channel_id: number
  channel_name: string
  health: ChannelMonitorHealth
  targets: ChannelMonitorTarget[]
  success_rate_24h: number | null
  samples_24h: number
  last_checked: number
}

type ChannelMonitorGroupAccumulator = ChannelMonitorGroup & {
  successful_samples_24h: number
}

const HEALTH_PRIORITY: Record<ChannelMonitorHealth, number> = {
  healthy: 0,
  degraded: 1,
  down: 2,
}

function mergeHealth(
  current: ChannelMonitorHealth,
  next: ChannelMonitorHealth
): ChannelMonitorHealth {
  return HEALTH_PRIORITY[next] > HEALTH_PRIORITY[current] ? next : current
}

export function groupChannelMonitorTargets(
  targets: ChannelMonitorTarget[]
): ChannelMonitorGroup[] {
  const groups = new Map<number, ChannelMonitorGroupAccumulator>()

  for (const target of targets) {
    let group = groups.get(target.channel_id)
    if (!group) {
      group = {
        channel_id: target.channel_id,
        channel_name: target.channel_name,
        health: target.health,
        targets: [],
        success_rate_24h: null,
        samples_24h: 0,
        last_checked: target.created_at,
        successful_samples_24h: 0,
      }
      groups.set(target.channel_id, group)
    }

    group.targets.push(target)
    group.health = mergeHealth(group.health, target.health)
    group.samples_24h += target.samples_24h
    group.successful_samples_24h += target.success_rate_24h * target.samples_24h
    group.last_checked = Math.max(group.last_checked, target.created_at)
  }

  return [...groups.values()]
    .sort((left, right) => left.channel_id - right.channel_id)
    .map((group) => {
      group.targets.sort((left, right) => left.model.localeCompare(right.model))
      return {
        channel_id: group.channel_id,
        channel_name: group.channel_name,
        health: group.health,
        targets: group.targets,
        samples_24h: group.samples_24h,
        success_rate_24h:
          group.samples_24h > 0
            ? group.successful_samples_24h / group.samples_24h
            : null,
        last_checked: group.last_checked,
      }
    })
}
