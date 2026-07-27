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
import { useQuery } from '@tanstack/react-query'
import type { PaginationState } from '@tanstack/react-table'
import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { useStatus } from '@/hooks/use-status'
import { isPersonalModeEnabled } from '@/lib/personal-mode'

import { channelsQueryKeys } from '../lib/channel-actions'
import { channelNeedsAttention } from '../lib/channel-attention'
import { fetchAllChannelPages } from '../lib/channel-pages'
import type { SearchChannelsParams } from '../types'

const ATTENTION_REFRESH_INTERVAL_MS = 5 * 60 * 1000

export function useChannelAttentionView(props: {
  params: SearchChannelsParams
  shouldSearch: boolean
  pagination: PaginationState
  resetPage: () => void
}) {
  const { t } = useTranslation()
  const { status } = useStatus()
  const personalMode = isPersonalModeEnabled(status)
  const [attentionOnly, setAttentionOnly] = useState(false)

  const query = useQuery({
    queryKey: [
      ...channelsQueryKeys.lists(),
      'personal-attention',
      props.shouldSearch,
      props.params,
    ],
    queryFn: () => fetchAllChannelPages(props.params, props.shouldSearch),
    enabled: personalMode,
    staleTime: ATTENTION_REFRESH_INTERVAL_MS,
    refetchInterval: personalMode ? ATTENTION_REFRESH_INTERVAL_MS : false,
    retry: false,
  })

  useEffect(() => {
    if (query.isError) {
      toast.error(t('Failed to refresh channel attention summary'))
    }
  }, [query.isError, t])

  const nowSeconds = Date.now() / 1000
  const attentionChannels = useMemo(
    () =>
      (query.data ?? []).filter((channel) =>
        channelNeedsAttention(channel, nowSeconds)
      ),
    [nowSeconds, query.data]
  )
  const pageChannels = useMemo(() => {
    const start = props.pagination.pageIndex * props.pagination.pageSize
    return attentionChannels.slice(start, start + props.pagination.pageSize)
  }, [attentionChannels, props.pagination])

  const changeMode = (nextAttentionOnly: boolean) => {
    setAttentionOnly(nextAttentionOnly)
    props.resetPage()
  }

  const refresh = () => {
    void query.refetch()
  }

  return {
    personalMode,
    attentionOnly: personalMode && attentionOnly,
    attentionCount: attentionChannels.length,
    channels: pageChannels,
    isLoading: query.isLoading,
    isFetching: query.isFetching,
    updatedAt: query.dataUpdatedAt,
    nowSeconds,
    changeMode,
    refresh,
  }
}
