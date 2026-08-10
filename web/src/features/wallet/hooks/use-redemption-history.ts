/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.
*/
import i18next from 'i18next'
import { useState, useEffect, useCallback } from 'react'
import { toast } from 'sonner'

import { getUserRedemptionHistory, isApiSuccess } from '../api'
import type { RedemptionHistoryRecord } from '../types'

export function useRedemptionHistory(
  options: { initialPage?: number; initialPageSize?: number } = {}
) {
  const { initialPage = 1, initialPageSize = 10 } = options
  const [records, setRecords] = useState<RedemptionHistoryRecord[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(initialPage)
  const [pageSize, setPageSize] = useState(initialPageSize)
  const [loading, setLoading] = useState(false)

  const fetchRedemptionHistory = useCallback(async () => {
    setLoading(true)
    try {
      const response = await getUserRedemptionHistory(page, pageSize)
      if (isApiSuccess(response) && response.data) {
        setRecords(response.data.items || [])
        setTotal(response.data.total || 0)
      } else {
        toast.error(
          response.message || i18next.t('Failed to load redemption history')
        )
        setRecords([])
        setTotal(0)
      }
    } catch (error) {
      // eslint-disable-next-line no-console
      console.error('Failed to fetch redemption history:', error)
      toast.error(i18next.t('Failed to load redemption history'))
      setRecords([])
      setTotal(0)
    } finally {
      setLoading(false)
    }
  }, [page, pageSize])

  useEffect(() => {
    fetchRedemptionHistory()
  }, [fetchRedemptionHistory])

  const handlePageChange = useCallback((newPage: number) => {
    setPage(newPage)
  }, [])

  const handlePageSizeChange = useCallback((newPageSize: number) => {
    setPageSize(newPageSize)
    setPage(1)
  }, [])

  return {
    records,
    total,
    page,
    pageSize,
    loading,
    handlePageChange,
    handlePageSizeChange,
    refresh: fetchRedemptionHistory,
  }
}
