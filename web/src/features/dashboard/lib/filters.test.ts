/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or (at your
option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { describe, expect, test } from 'vitest'

import {
  DEFAULT_DASHBOARD_CHART_PREFERENCES,
  TIME_RANGE_PRESETS,
} from '../constants'
import { buildDefaultDashboardFilters, getDashboardDateRange } from './filters'

describe('dashboard date ranges', () => {
  test('uses the current calendar day for the Today preset', () => {
    const reference = new Date(2026, 7, 31, 15, 30, 45, 123)
    const { start, end } = getDashboardDateRange(0, reference)

    expect(start).toEqual(new Date(2026, 7, 31, 0, 0, 0, 0))
    expect(end).toEqual(new Date(2026, 7, 31, 23, 59, 59, 999))
  })

  test('keeps one day as a rolling 24-hour range', () => {
    const reference = new Date(2026, 7, 31, 15, 30, 45, 123)
    const { start, end } = getDashboardDateRange(1, reference)

    expect(start).toEqual(new Date(2026, 7, 30, 15, 30, 45, 123))
    expect(end).toEqual(reference)
  })

  test('defaults model analytics to Today and removes the 29-day preset', () => {
    expect(DEFAULT_DASHBOARD_CHART_PREFERENCES.defaultTimeRangeDays).toBe(0)
    expect(TIME_RANGE_PRESETS.map((preset) => preset.days)).toEqual([
      0, 1, 7, 14,
    ])

    const reference = new Date()
    const expected = getDashboardDateRange(0, reference)
    const filters = buildDefaultDashboardFilters(
      DEFAULT_DASHBOARD_CHART_PREFERENCES
    )

    expect(filters.start_timestamp?.getHours()).toBe(0)
    expect(filters.start_timestamp?.getMinutes()).toBe(0)
    expect(filters.end_timestamp?.getHours()).toBe(23)
    expect(filters.end_timestamp?.getMinutes()).toBe(59)
    expect(filters.start_timestamp?.toDateString()).toBe(
      expected.start.toDateString()
    )
    expect(filters.end_timestamp?.toDateString()).toBe(
      expected.end.toDateString()
    )
  })
})
