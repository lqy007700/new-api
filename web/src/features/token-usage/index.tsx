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
import { keepPreviousData, useQuery } from '@tanstack/react-query'
import type { PaginationState, Updater } from '@tanstack/react-table'
import { RefreshCw } from 'lucide-react'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { DataTablePage, useDataTable } from '@/components/data-table'
import { SectionPageLayout } from '@/components/layout'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { formatBillingCurrencyFromUSD } from '@/lib/currency'

import { getTokenUsage } from './api'
import { useTokenUsageColumns } from './components/token-usage-columns'
import {
  defaultTokenUsageRange,
  validateTokenUsageRange,
} from './lib/usage'
import type {
  TokenUsageGroupBy,
  TokenUsageQuery,
  TokenUsageSummary,
} from './types'

const DEFAULT_PAGE_SIZE = 20

export function TokenUsage() {
  const { t } = useTranslation()
  const defaults = useMemo(defaultTokenUsageRange, [])
  const [from, setFrom] = useState(defaults.from)
  const [to, setTo] = useState(defaults.to)
  const [tokenName, setTokenName] = useState('')
  const [groupBy, setGroupBy] = useState<TokenUsageGroupBy>('detail')
  const [appliedFilters, setAppliedFilters] = useState({
    ...defaults,
    tokenName: '',
    groupBy: 'detail' as TokenUsageGroupBy,
  })
  const [pagination, setPagination] = useState<PaginationState>({
    pageIndex: 0,
    pageSize: DEFAULT_PAGE_SIZE,
  })

  const query: TokenUsageQuery = {
    ...appliedFilters,
    page: pagination.pageIndex + 1,
    pageSize: pagination.pageSize,
  }
  const rangeIsValid = validateTokenUsageRange(from, to)
  const columns = useTokenUsageColumns(appliedFilters.groupBy)
  const usageQuery = useQuery({
    queryKey: ['token-usage', query],
    queryFn: () => getTokenUsage(query),
    staleTime: 60_000,
    placeholderData: keepPreviousData,
  })

  const data = usageQuery.data?.data
  const rows = data?.page.items ?? []
  const summary = data?.summary

  const { table } = useDataTable({
    data: rows,
    columns,
    columnFilters: [],
    globalFilter: '',
    pagination,
    onPaginationChange: handlePaginationChange,
    manualPagination: true,
    totalCount: data?.page.total ?? 0,
    getRowId: (row) => `${row.usage_date}-${row.token_id}`,
  })

  const hasDraftFilters =
    from !== defaults.from ||
    to !== defaults.to ||
    tokenName.trim() !== '' ||
    groupBy !== 'detail'
  const hasAppliedFilters =
    appliedFilters.from !== defaults.from ||
    appliedFilters.to !== defaults.to ||
    appliedFilters.tokenName !== '' ||
    appliedFilters.groupBy !== 'detail'

  function handlePaginationChange(updater: Updater<PaginationState>) {
    setPagination((current) =>
      typeof updater === 'function' ? updater(current) : updater
    )
  }

  function applyFilters() {
    if (!rangeIsValid) return
    setAppliedFilters({ from, to, tokenName: tokenName.trim(), groupBy })
    setPagination((current) => ({ ...current, pageIndex: 0 }))
  }

  function resetFilters() {
    setFrom(defaults.from)
    setTo(defaults.to)
    setTokenName('')
    setGroupBy('detail')
    setAppliedFilters({
      ...defaults,
      tokenName: '',
      groupBy: 'detail',
    })
    setPagination((current) => ({ ...current, pageIndex: 0 }))
  }

  return (
    <SectionPageLayout fixedContent>
      <SectionPageLayout.Title>{t('API Usage')}</SectionPageLayout.Title>
      <SectionPageLayout.Actions>
        <Button
          variant='outline'
          onClick={() => usageQuery.refetch()}
          disabled={usageQuery.isFetching}
        >
          <RefreshCw className={usageQuery.isFetching ? 'animate-spin' : ''} />
          {t('Refresh')}
        </Button>
      </SectionPageLayout.Actions>
      <SectionPageLayout.Content>
        <div className='flex h-full min-h-0 flex-col gap-3'>
          {!rangeIsValid && (
            <Alert variant='destructive'>
              <AlertDescription>
                {t('Select a valid date range of no more than 366 days.')}
              </AlertDescription>
            </Alert>
          )}
          {usageQuery.isError && (
            <Alert variant='destructive'>
              <AlertDescription>{usageQuery.error.message}</AlertDescription>
            </Alert>
          )}
          <DataTablePage
            table={table}
            columns={columns}
            isLoading={usageQuery.isLoading}
            isFetching={usageQuery.isFetching}
            emptyTitle={t('No usage data in the selected range.')}
            emptyDescription={t(
              'Try changing the date range or API key name and search again.'
            )}
            skeletonKeyPrefix='token-usage-skeleton'
            applyHeaderSize
            toolbarProps={{
              customSearch: (
                <Input
                  value={tokenName}
                  placeholder={t('Search by API key name')}
                  aria-label={t('Search by API key name')}
                  onChange={(event) => setTokenName(event.target.value)}
                  onKeyDown={(event) => {
                    if (event.key === 'Enter') applyFilters()
                  }}
                  className='w-full sm:w-50 lg:w-60'
                />
              ),
              additionalSearch: (
                <>
                  <Input
                    type='date'
                    value={from}
                    aria-label={t('Start date')}
                    title={t('Start date')}
                    onChange={(event) => setFrom(event.target.value)}
                    className='w-full sm:w-38'
                  />
                  <span className='text-muted-foreground hidden sm:inline'>–</span>
                  <Input
                    type='date'
                    max={defaults.to}
                    value={to}
                    aria-label={t('End date')}
                    title={t('End date')}
                    onChange={(event) => setTo(event.target.value)}
                    className='w-full sm:w-38'
                  />
                  <Select
                    value={groupBy}
                    onValueChange={(value) =>
                      setGroupBy(value as TokenUsageGroupBy)
                    }
                  >
                    <SelectTrigger className='w-full sm:w-32'>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectGroup>
                        <SelectItem value='detail'>{t('Details')}</SelectItem>
                        <SelectItem value='daily'>{t('By day')}</SelectItem>
                        <SelectItem value='token'>{t('By API key')}</SelectItem>
                      </SelectGroup>
                    </SelectContent>
                  </Select>
                </>
              ),
              hasAdditionalFilters: hasDraftFilters || hasAppliedFilters,
              onReset: resetFilters,
              onSearch: applyFilters,
              searchLoading: usageQuery.isFetching,
              hideViewOptions: true,
            }}
            afterTable={<UsageSummary summary={summary} />}
          />
        </div>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}

function UsageSummary({ summary }: { summary: TokenUsageSummary | undefined }) {
  const { t } = useTranslation()
  const items = [
    { label: t('Requests'), value: formatNumber(summary?.request_count) },
    { label: t('Input Tokens'), value: formatNumber(summary?.input_tokens) },
    { label: t('Output Tokens'), value: formatNumber(summary?.output_tokens) },
    { label: t('Cached'), value: formatNumber(summary?.cache_tokens) },
    { label: t('Total Tokens'), value: formatNumber(summary?.total_tokens) },
    { label: t('Cost'), value: formatUsageCost(summary?.cost ?? 0) },
  ]

  return (
    <div className='bg-muted/30 grid grid-cols-2 gap-px overflow-hidden rounded-lg border sm:grid-cols-3 xl:grid-cols-6'>
      {items.map((item) => (
        <div key={item.label} className='bg-background px-3 py-2'>
          <div className='text-muted-foreground text-xs'>{item.label}</div>
          <div className='mt-0.5 text-sm font-semibold tabular-nums'>
            {item.value}
          </div>
        </div>
      ))}
    </div>
  )
}

function formatNumber(value: number | undefined): string {
  return (value ?? 0).toLocaleString()
}

function formatUsageCost(value: number): string {
  return formatBillingCurrencyFromUSD(value, {
    digitsLarge: 6,
    digitsSmall: 6,
    abbreviate: false,
  })
}
