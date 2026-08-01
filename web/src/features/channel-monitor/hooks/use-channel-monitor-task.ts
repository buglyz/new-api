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
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useCallback, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { getChannelMonitorTask } from '../api'

function isActiveTaskStatus(status?: string) {
  return status === 'pending' || status === 'running'
}

export function useChannelMonitorTask() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [taskID, setTaskID] = useState<string | null>(null)
  const taskQuery = useQuery({
    queryKey: ['channel-monitor-task', taskID],
    queryFn: () => {
      if (!taskID) return Promise.reject(new Error('monitor task is required'))
      return getChannelMonitorTask(taskID)
    },
    enabled: Boolean(taskID),
    refetchInterval: (query) => {
      if (
        query.state.error ||
        (query.state.data &&
          (!query.state.data.success || !query.state.data.data))
      ) {
        return 5000
      }
      const status = query.state.data?.data?.status
      return isActiveTaskStatus(status) ? 1000 : false
    },
    retry: 3,
  })

  useEffect(() => {
    if (!taskID) return
    const response = taskQuery.data
    if (
      taskQuery.isError ||
      (response && (!response.success || !response.data))
    ) {
      return
    }

    const task = response?.data
    if (!task || isActiveTaskStatus(task.status)) return

    setTaskID(null)
    if (task.status === 'succeeded') {
      toast.success(t('Monitor run completed'))
    } else {
      toast.error(task.error || t('Monitor run failed'))
    }
    void queryClient.invalidateQueries({ queryKey: ['channel-monitor'] })
    void queryClient.invalidateQueries({
      queryKey: ['channel-monitor-history'],
    })
  }, [queryClient, taskID, taskQuery.data, taskQuery.isError, t])

  const watchTask = useCallback((nextTaskID: string) => {
    setTaskID(nextTaskID)
  }, [])

  return {
    isWatchingTask: Boolean(taskID),
    isStatusUnknown: Boolean(
      taskID &&
      (taskQuery.isError ||
        (taskQuery.data && (!taskQuery.data.success || !taskQuery.data.data)))
    ),
    watchTask,
  }
}
