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
import dayjs from '@/lib/dayjs'

import type { TokenUsageGroupBy } from '../types'

export function defaultTokenUsageRange(): { from: string; to: string } {
  const to = dayjs().subtract(1, 'day')
  return {
    from: to.subtract(29, 'day').format('YYYY-MM-DD'),
    to: to.format('YYYY-MM-DD'),
  }
}

export function validateTokenUsageRange(from: string, to: string): boolean {
  if (!/^\d{4}-\d{2}-\d{2}$/.test(from) || !/^\d{4}-\d{2}-\d{2}$/.test(to)) {
    return false
  }
  const fromDate = dayjs(from)
  const toDate = dayjs(to)
  if (
    !fromDate.isValid() ||
    !toDate.isValid() ||
    fromDate.format('YYYY-MM-DD') !== from ||
    toDate.format('YYYY-MM-DD') !== to ||
    toDate.isBefore(fromDate)
  ) {
    return false
  }
  return toDate.diff(fromDate, 'day') < 366
}

export function showUsageDate(groupBy: TokenUsageGroupBy): boolean {
  return groupBy !== 'token'
}

export function showUsageToken(groupBy: TokenUsageGroupBy): boolean {
  return groupBy !== 'daily'
}

export function formatUsageDate(value: number): string {
  const date = String(value)
  if (date.length !== 8) return '-'
  return `${date.slice(0, 4)}-${date.slice(4, 6)}-${date.slice(6, 8)}`
}
