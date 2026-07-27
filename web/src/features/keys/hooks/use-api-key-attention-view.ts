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

import { apiKeyNeedsAttention } from '../lib/api-key-attention'
import { fetchAllApiKeyPages } from '../lib/api-key-pages'

export function useApiKeyAttentionView(props: {
  keyword: string | undefined
  token: string
  status: string | undefined
  refreshTrigger: number
  pagination: PaginationState
  resetPage: () => void
}) {
  const { t } = useTranslation()
  const { status } = useStatus()
  const personalMode = isPersonalModeEnabled(status)
  const [attentionOnly, setAttentionOnly] = useState(false)
  const query = useQuery({
    queryKey: [
      'keys',
      'personal-attention',
      props.keyword,
      props.token,
      props.refreshTrigger,
    ],
    queryFn: () =>
      fetchAllApiKeyPages({ keyword: props.keyword, token: props.token }),
    enabled: personalMode,
    staleTime: 5 * 60 * 1000,
    retry: false,
  })

  useEffect(() => {
    if (query.isError) toast.error(t('Failed to load API keys'))
  }, [query.isError, t])

  const nowSeconds = Date.now() / 1000
  const attentionKeys = useMemo(
    () =>
      (query.data ?? []).filter(
        (apiKey) =>
          (!props.status || String(apiKey.status) === props.status) &&
          apiKeyNeedsAttention(apiKey, nowSeconds)
      ),
    [nowSeconds, props.status, query.data]
  )
  const pageKeys = useMemo(() => {
    const start = props.pagination.pageIndex * props.pagination.pageSize
    return attentionKeys.slice(start, start + props.pagination.pageSize)
  }, [attentionKeys, props.pagination])

  const changeMode = (nextAttentionOnly: boolean) => {
    setAttentionOnly(nextAttentionOnly)
    props.resetPage()
  }

  return {
    personalMode,
    attentionOnly: personalMode && attentionOnly,
    attentionCount: attentionKeys.length,
    keys: pageKeys,
    isLoading: query.isLoading,
    isFetching: query.isFetching,
    nowSeconds,
    changeMode,
  }
}
