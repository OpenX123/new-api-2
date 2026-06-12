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
import { api } from '@/lib/api'
import type {
  AdminInvoice,
  ApiResponse,
  GetInvoicesParams,
  GetInvoicesResponse,
} from './types'

// ============================================================================
// Admin Invoice Management API
// ============================================================================

// Get paginated admin invoice list
export async function getInvoices(
  params: GetInvoicesParams = {}
): Promise<GetInvoicesResponse> {
  const { p = 1, page_size = 20, status, keyword } = params
  const searchParams = new URLSearchParams({
    p: String(p),
    page_size: String(page_size),
  })
  if (status !== undefined && status !== 0) {
    searchParams.set('status', String(status))
  }
  if (keyword) {
    searchParams.set('keyword', keyword)
  }
  const res = await api.get(`/api/invoice/?${searchParams.toString()}`)
  return res.data
}

// Get single invoice detail with orders
export async function getInvoice(
  id: number
): Promise<ApiResponse<AdminInvoice>> {
  const res = await api.get(`/api/invoice/${id}`)
  return res.data
}

// Upload invoice file (pdf/png/jpg ≤10MB) and complete the invoice
export async function uploadInvoiceFile(
  id: number,
  file: File
): Promise<ApiResponse> {
  const formData = new FormData()
  formData.append('file', file)
  const res = await api.post(`/api/invoice/${id}/file`, formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
  })
  return res.data
}

// Reject an invoice with a reason
export async function rejectInvoice(
  id: number,
  reason: string
): Promise<ApiResponse> {
  const res = await api.post(`/api/invoice/${id}/reject`, { reason })
  return res.data
}

// Download the uploaded invoice file
export async function downloadInvoiceFileAdmin(
  id: number,
  invoiceNo: string
): Promise<void> {
  const res = await api.get(`/api/invoice/${id}/file`, {
    responseType: 'blob',
  })
  const mime = (res.headers['content-type'] as string | undefined) || ''
  const ext = mime.includes('png')
    ? '.png'
    : mime.includes('jpeg')
      ? '.jpg'
      : '.pdf'
  const url = URL.createObjectURL(res.data as Blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `invoice-${invoiceNo}${ext}`
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
}
