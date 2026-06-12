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

// ============================================================================
// Invoice Management Types (Admin)
// ============================================================================

export interface AdminInvoiceOrder {
  id: number
  order_type: string
  order_id: string
  trade_no: string
  money: number
}

export interface AdminInvoice {
  id: number
  user_id: number
  username: string
  invoice_no: string
  title_type: 1 | 2 // 1 = personal, 2 = company
  title_name: string
  tax_no: string
  email: string
  money: number
  status: 1 | 2 | 3 // 1 = pending, 2 = completed, 3 = rejected
  reject_reason: string
  remark: string
  create_time: number
  complete_time: number
  has_file: boolean
  orders?: AdminInvoiceOrder[]
}

// ============================================================================
// API Request/Response Types
// ============================================================================

export interface ApiResponse<T = unknown> {
  success: boolean
  message?: string
  data?: T
}

export interface GetInvoicesParams {
  p?: number
  page_size?: number
  status?: number
  keyword?: string
}

export interface GetInvoicesResponse {
  success: boolean
  message?: string
  data?: {
    items: AdminInvoice[]
    total: number
  }
}
