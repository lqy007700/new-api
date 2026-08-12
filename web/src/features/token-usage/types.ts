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
export type TokenUsageGroupBy = 'detail' | 'daily' | 'token'

export type TokenUsageRow = {
  usage_date: number
  token_id: number
  token_name: string
  request_count: number
  input_tokens: number
  output_tokens: number
  cache_tokens: number
  total_tokens: number
  quota: number
  cost: number
}

export type TokenUsageSummary = Omit<
  TokenUsageRow,
  'usage_date' | 'token_id' | 'token_name'
>

export type TokenUsageResponse = {
  success: boolean
  message: string
  data: {
    page: {
      page: number
      page_size: number
      total: number
      items: TokenUsageRow[]
    }
    summary: TokenUsageSummary
  }
}

export type TokenUsageQuery = {
  from: string
  to: string
  tokenName: string
  groupBy: TokenUsageGroupBy
  page: number
  pageSize: number
}
