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
import { describe, expect, test } from 'vitest'

import {
  formatUsageDate,
  showUsageDate,
  showUsageToken,
  validateTokenUsageRange,
} from '../usage'

describe('token usage date range validation', () => {
  test('accepts an inclusive range of up to 366 days', () => {
    expect(validateTokenUsageRange('2025-08-12', '2026-08-11')).toBe(true)
  })

  test('rejects invalid, reversed, and overly broad ranges', () => {
    expect(validateTokenUsageRange('2026-02-30', '2026-03-01')).toBe(false)
    expect(validateTokenUsageRange('2026-08-12', '2026-08-11')).toBe(false)
    expect(validateTokenUsageRange('2025-08-11', '2026-08-12')).toBe(false)
  })
})

describe('token usage presentation modes', () => {
  test('shows only fields relevant to each aggregation mode', () => {
    expect(showUsageDate('detail')).toBe(true)
    expect(showUsageToken('detail')).toBe(true)
    expect(showUsageDate('daily')).toBe(true)
    expect(showUsageToken('daily')).toBe(false)
    expect(showUsageDate('token')).toBe(false)
    expect(showUsageToken('token')).toBe(true)
  })

  test('formats integer day keys without timezone conversion', () => {
    expect(formatUsageDate(20260811)).toBe('2026-08-11')
    expect(formatUsageDate(0)).toBe('-')
  })
})
