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
import { useRef, useState } from 'react'
import { Download, Upload } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Textarea } from '@/components/ui/textarea'
import {
  downloadInvoiceFileAdmin,
  getInvoice,
  rejectInvoice,
  uploadInvoiceFile,
} from '../api'
import type { AdminInvoice } from '../types'

interface InvoiceDetailDrawerProps {
  invoiceId: number | null
  open: boolean
  onOpenChange: (open: boolean) => void
  onChanged: () => void
}

const MAX_FILE_SIZE = 10 * 1024 * 1024 // 10MB

export function InvoiceDetailDrawer({
  invoiceId,
  open,
  onOpenChange,
  onChanged,
}: InvoiceDetailDrawerProps) {
  const { t } = useTranslation()
  const [invoice, setInvoice] = useState<AdminInvoice | null>(null)
  const [loading, setLoading] = useState(false)
  const [rejectReason, setRejectReason] = useState('')
  const [actionLoading, setActionLoading] = useState(false)
  const fileInputRef = useRef<HTMLInputElement>(null)

  // Load invoice detail when drawer opens
  const handleOpenChange = async (newOpen: boolean) => {
    onOpenChange(newOpen)
    if (newOpen && invoiceId != null) {
      setLoading(true)
      setRejectReason('')
      try {
        const res = await getInvoice(invoiceId)
        if (res.success && res.data) {
          setInvoice(res.data)
        } else {
          toast.error(res.message ?? t('Failed to load invoice'))
        }
      } catch {
        toast.error(t('Failed to load invoice'))
      } finally {
        setLoading(false)
      }
    } else if (!newOpen) {
      setInvoice(null)
    }
  }

  const handleFileUpload = async (file: File) => {
    if (!invoice) return
    if (file.size > MAX_FILE_SIZE) {
      toast.error(t('File size must not exceed 10MB'))
      return
    }
    setActionLoading(true)
    try {
      const res = await uploadInvoiceFile(invoice.id, file)
      if (res.success) {
        toast.success(t('Invoice file uploaded successfully'))
        onChanged()
        onOpenChange(false)
      } else {
        toast.error(res.message ?? t('Upload failed'))
      }
    } catch {
      toast.error(t('Upload failed'))
    } finally {
      setActionLoading(false)
    }
  }

  const handleFileInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (file) {
      void handleFileUpload(file)
    }
    // reset so same file can be re-selected
    e.target.value = ''
  }

  const handleReject = async () => {
    if (!invoice) return
    if (!rejectReason.trim()) {
      toast.error(t('Please enter a reject reason'))
      return
    }
    setActionLoading(true)
    try {
      const res = await rejectInvoice(invoice.id, rejectReason.trim())
      if (res.success) {
        toast.success(t('Invoice rejected'))
        onChanged()
        onOpenChange(false)
      } else {
        toast.error(res.message ?? t('Reject failed'))
      }
    } catch {
      toast.error(t('Reject failed'))
    } finally {
      setActionLoading(false)
    }
  }

  const handleDownload = async () => {
    if (!invoice) return
    setActionLoading(true)
    try {
      await downloadInvoiceFileAdmin(invoice.id, invoice.invoice_no)
    } catch {
      toast.error(t('Download failed'))
    } finally {
      setActionLoading(false)
    }
  }

  const getStatusLabel = (status: number) => {
    switch (status) {
      case 1:
        return t('Pending Invoice')
      case 2:
        return t('Invoiced')
      case 3:
        return t('Rejected')
      default:
        return t('Unknown')
    }
  }

  const getStatusVariant = (
    status: number
  ): 'default' | 'secondary' | 'destructive' | 'outline' => {
    switch (status) {
      case 1:
        return 'secondary'
      case 2:
        return 'default'
      case 3:
        return 'destructive'
      default:
        return 'outline'
    }
  }

  return (
    <Sheet open={open} onOpenChange={handleOpenChange}>
      <SheetContent className='w-full overflow-y-auto sm:max-w-xl'>
        <SheetHeader>
          <SheetTitle>{t('Invoice Detail')}</SheetTitle>
        </SheetHeader>

        {loading && (
          <div className='flex items-center justify-center py-12'>
            <span className='text-muted-foreground text-sm'>
              {t('Loading...')}
            </span>
          </div>
        )}

        {!loading && invoice && (
          <div className='space-y-6 px-1 py-4'>
            {/* Shared hidden file input for upload actions */}
            <input
              ref={fileInputRef}
              type='file'
              accept='.pdf,.png,.jpg,.jpeg'
              className='hidden'
              onChange={handleFileInputChange}
            />
            {/* Header Info */}
            <div className='flex items-center justify-between'>
              <span className='text-muted-foreground font-mono text-sm'>
                {invoice.invoice_no}
              </span>
              <Badge variant={getStatusVariant(invoice.status)}>
                {getStatusLabel(invoice.status)}
              </Badge>
            </div>

            {/* Title Info Grid */}
            <div className='space-y-2'>
              <h3 className='text-sm font-semibold'>
                {t('Invoice Title Information')}
              </h3>
              <div className='grid grid-cols-2 gap-x-4 gap-y-2 text-sm'>
                <div className='text-muted-foreground'>{t('User')}</div>
                <div>{invoice.username}</div>

                <div className='text-muted-foreground'>{t('Title Type')}</div>
                <div>
                  {invoice.title_type === 1
                    ? t('Personal')
                    : t('Enterprise')}
                </div>

                <div className='text-muted-foreground'>{t('Title Name')}</div>
                <div>{invoice.title_name}</div>

                {invoice.title_type === 2 && (
                  <>
                    <div className='text-muted-foreground'>
                      {t('Tax Number')}
                    </div>
                    <div>{invoice.tax_no || '-'}</div>
                  </>
                )}

                <div className='text-muted-foreground'>{t('Email')}</div>
                <div>{invoice.email}</div>

                <div className='text-muted-foreground'>{t('Amount')}</div>
                <div className='font-medium'>
                  ${invoice.money.toFixed(2)}
                </div>
              </div>
            </div>

            {/* Remark */}
            {invoice.remark && (
              <div className='space-y-1'>
                <h3 className='text-sm font-semibold'>{t('Remark')}</h3>
                <p className='text-muted-foreground text-sm'>
                  {invoice.remark}
                </p>
              </div>
            )}

            {/* Reject Reason (status 3) */}
            {invoice.status === 3 && invoice.reject_reason && (
              <div className='space-y-1'>
                <h3 className='text-sm font-semibold text-red-600'>
                  {t('Reject Reason')}
                </h3>
                <p className='text-sm text-red-600'>{invoice.reject_reason}</p>
              </div>
            )}

            {/* Related Orders */}
            {invoice.orders && invoice.orders.length > 0 && (
              <div className='space-y-2'>
                <h3 className='text-sm font-semibold'>
                  {t('Related Orders')}
                </h3>
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead className='text-xs'>
                        {t('Order Type')}
                      </TableHead>
                      <TableHead className='text-xs'>
                        {t('Trade No')}
                      </TableHead>
                      <TableHead className='text-right text-xs'>
                        {t('Amount')}
                      </TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {invoice.orders.map((order) => (
                      <TableRow key={order.id}>
                        <TableCell className='text-xs'>
                          {order.order_type}
                        </TableCell>
                        <TableCell className='font-mono text-xs'>
                          {order.trade_no}
                        </TableCell>
                        <TableCell className='text-right text-xs'>
                          ${order.money.toFixed(2)}
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            )}

            {/* Actions: Pending (status 1) */}
            {invoice.status === 1 && (
              <div className='space-y-4 border-t pt-4'>
                {/* Upload Section */}
                <div className='space-y-2'>
                  <Button
                    className='w-full'
                    disabled={actionLoading}
                    onClick={() => fileInputRef.current?.click()}
                  >
                    <Upload className='mr-2 h-4 w-4' />
                    {t('Upload Invoice File')}
                  </Button>
                  <p className='text-muted-foreground text-xs'>
                    {t('Accept: PDF, PNG, JPG (max 10MB)')}
                  </p>
                </div>

                {/* Reject Section */}
                <div className='space-y-2'>
                  <Label htmlFor='reject-reason'>{t('Reject Reason')}</Label>
                  <Textarea
                    id='reject-reason'
                    placeholder={t('Enter reject reason...')}
                    value={rejectReason}
                    onChange={(e) => setRejectReason(e.target.value)}
                    rows={3}
                  />
                  <Button
                    variant='destructive'
                    className='w-full'
                    disabled={actionLoading || !rejectReason.trim()}
                    onClick={handleReject}
                  >
                    {t('Reject Invoice')}
                  </Button>
                </div>
              </div>
            )}

            {/* Actions: Completed (status 2) */}
            {invoice.status === 2 && (
              <div className='space-y-2 border-t pt-4'>
                {invoice.has_file && (
                  <Button
                    variant='outline'
                    className='w-full'
                    disabled={actionLoading}
                    onClick={handleDownload}
                  >
                    <Download className='mr-2 h-4 w-4' />
                    {t('Download Invoice File')}
                  </Button>
                )}
                <Button
                  variant='outline'
                  className='w-full'
                  disabled={actionLoading}
                  onClick={() => fileInputRef.current?.click()}
                >
                  <Upload className='mr-2 h-4 w-4' />
                  {t('Re-upload Invoice File')}
                </Button>
              </div>
            )}
          </div>
        )}
      </SheetContent>
    </Sheet>
  )
}
