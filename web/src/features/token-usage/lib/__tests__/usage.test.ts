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
import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import {
  formatUsageDate,
  showUsageDate,
  showUsageToken,
  validateTokenUsageRange,
} from '../usage'

describe('token usage date range validation', () => {
  test('accepts an inclusive range of up to 366 days', () => {
    assert.equal(validateTokenUsageRange('2025-08-12', '2026-08-11'), true)
  })

  test('rejects invalid, reversed, and overly broad ranges', () => {
    assert.equal(validateTokenUsageRange('2026-02-30', '2026-03-01'), false)
    assert.equal(validateTokenUsageRange('2026-08-12', '2026-08-11'), false)
    assert.equal(validateTokenUsageRange('2025-08-11', '2026-08-12'), false)
  })
})

describe('token usage presentation modes', () => {
  test('shows only fields relevant to each aggregation mode', () => {
    assert.equal(showUsageDate('detail'), true)
    assert.equal(showUsageToken('detail'), true)
    assert.equal(showUsageDate('daily'), true)
    assert.equal(showUsageToken('daily'), false)
    assert.equal(showUsageDate('token'), false)
    assert.equal(showUsageToken('token'), true)
  })

  test('formats integer day keys without timezone conversion', () => {
    assert.equal(formatUsageDate(20260811), '2026-08-11')
    assert.equal(formatUsageDate(0), '-')
  })
})
