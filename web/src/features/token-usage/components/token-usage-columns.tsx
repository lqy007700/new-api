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
import type { ColumnDef } from '@tanstack/react-table'
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'

import { formatBillingCurrencyFromUSD } from '@/lib/currency'

import { formatUsageDate, showUsageDate, showUsageToken } from '../lib/usage'
import type { TokenUsageGroupBy, TokenUsageRow } from '../types'

export function useTokenUsageColumns(
  groupBy: TokenUsageGroupBy
): ColumnDef<TokenUsageRow, unknown>[] {
  const { t } = useTranslation()

  return useMemo(() => {
    const columns: ColumnDef<TokenUsageRow, unknown>[] = []

    if (showUsageDate(groupBy)) {
      columns.push({
        id: 'usage_date',
        accessorKey: 'usage_date',
        header: t('Date'),
        cell: ({ row }) => formatUsageDate(row.original.usage_date),
        size: 120,
        meta: {
          label: t('Date'),
          mobileTitle: groupBy === 'daily',
          mobileOrder: 10,
        },
      })
    }

    if (showUsageToken(groupBy)) {
      columns.push({
        id: 'token_name',
        accessorKey: 'token_name',
        header: t('API Key'),
        cell: ({ row }) => (
          <div className='min-w-0'>
            <div
              className='max-w-72 truncate font-medium'
              title={row.original.token_name}
            >
              {row.original.token_name || t('Unassigned')}
            </div>
            {row.original.token_id > 0 && (
              <div className='text-muted-foreground text-xs'>
                ID {row.original.token_id}
              </div>
            )}
          </div>
        ),
        minSize: 180,
        meta: {
          label: t('API Key'),
          mobileTitle: true,
          pinned: 'left',
        },
      })
    }

    columns.push(
      numberColumn('request_count', t('Requests'), 100, 20),
      numberColumn('input_tokens', t('Input Tokens'), 120, 30),
      numberColumn('output_tokens', t('Output Tokens'), 120, 40),
      numberColumn('cache_tokens', t('Cached'), 110, 50),
      numberColumn('total_tokens', t('Total Tokens'), 120, 60),
      {
        id: 'cost',
        accessorKey: 'cost',
        header: () => <div className='text-right'>{t('Cost')}</div>,
        cell: ({ row }) => (
          <div className='text-right font-medium tabular-nums'>
            {formatUsageCost(row.original.cost)}
          </div>
        ),
        size: 120,
        meta: { label: t('Cost'), mobileOrder: 70 },
      }
    )

    return columns
  }, [groupBy, t])
}

function numberColumn(
  key: keyof TokenUsageRow,
  label: string,
  size: number,
  mobileOrder: number
): ColumnDef<TokenUsageRow, unknown> {
  return {
    id: key,
    accessorKey: key,
    header: () => <div className='text-right'>{label}</div>,
    cell: ({ row }) => (
      <div className='text-right tabular-nums'>
        {(row.original[key] as number).toLocaleString()}
      </div>
    ),
    size,
    meta: { label, mobileOrder },
  }
}

function formatUsageCost(value: number): string {
  return formatBillingCurrencyFromUSD(value, {
    digitsLarge: 6,
    digitsSmall: 6,
    abbreviate: false,
  })
}
