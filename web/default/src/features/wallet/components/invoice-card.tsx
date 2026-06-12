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
import { useState, useEffect, useCallback } from 'react'
import { Download, Loader2, X } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Checkbox } from '@/components/ui/checkbox'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  getInvoiceableOrders,
  getMyInvoices,
  cancelInvoice,
  downloadInvoiceFile,
  isApiSuccess,
} from '../api'
import type {
  InvoiceableOrder,
  InvoiceTitle,
  InvoiceRecord,
} from '../types'
import { InvoiceApplyDialog } from './dialogs/invoice-apply-dialog'

export function InvoiceCard() {
  const { t } = useTranslation()

  const [invoiceableOrders, setInvoiceableOrders] = useState<
    InvoiceableOrder[]
  >([])
  const [defaultTitle, setDefaultTitle] = useState<InvoiceTitle | null>(null)
  const [invoiceRecords, setInvoiceRecords] = useState<InvoiceRecord[]>([])
  const [loading, setLoading] = useState(true)
  const [selectedOrderIds, setSelectedOrderIds] = useState<Set<string>>(
    new Set()
  )
  const [applyDialogOpen, setApplyDialogOpen] = useState(false)
  const [cancellingId, setCancellingId] = useState<number | null>(null)
  const [downloadingId, setDownloadingId] = useState<number | null>(null)

  const refresh = useCallback(async () => {
    setLoading(true)
    try {
      const [ordersRes, recordsRes] = await Promise.all([
        getInvoiceableOrders(),
        getMyInvoices(1, 50),
      ])
      if (isApiSuccess(ordersRes) && ordersRes.data) {
        setInvoiceableOrders(ordersRes.data.orders ?? [])
        setDefaultTitle(ordersRes.data.default_title ?? null)
      }
      if (isApiSuccess(recordsRes) && recordsRes.data) {
        setInvoiceRecords(recordsRes.data.items ?? [])
      }
    } catch {
      toast.error(t('Failed to load invoice data'))
    } finally {
      setLoading(false)
      setSelectedOrderIds(new Set())
    }
  }, [t])

  useEffect(() => {
    void refresh()
  }, [refresh])

  const selectedOrders = invoiceableOrders.filter((o) =>
    selectedOrderIds.has(`${o.order_type}-${o.order_id}`)
  )

  const toggleOrder = (order: InvoiceableOrder) => {
    const key = `${order.order_type}-${order.order_id}`
    setSelectedOrderIds((prev) => {
      const next = new Set(prev)
      if (next.has(key)) {
        next.delete(key)
      } else {
        next.add(key)
      }
      return next
    })
  }

  const toggleAll = () => {
    if (selectedOrderIds.size === invoiceableOrders.length) {
      setSelectedOrderIds(new Set())
    } else {
      setSelectedOrderIds(
        new Set(invoiceableOrders.map((o) => `${o.order_type}-${o.order_id}`))
      )
    }
  }

  const handleCancel = async (record: InvoiceRecord) => {
    setCancellingId(record.id)
    try {
      const res = await cancelInvoice(record.id)
      if (isApiSuccess(res)) {
        toast.success(t('Invoice application cancelled'))
        await refresh()
      } else {
        toast.error(res.message || t('Failed to cancel invoice application'))
      }
    } catch {
      toast.error(t('Failed to cancel invoice application'))
    } finally {
      setCancellingId(null)
    }
  }

  const handleDownload = async (record: InvoiceRecord) => {
    setDownloadingId(record.id)
    try {
      await downloadInvoiceFile(record.id, record.invoice_no)
    } catch {
      toast.error(t('Failed to download invoice file'))
    } finally {
      setDownloadingId(null)
    }
  }

  const getOrderTypeLabel = (type: 'topup' | 'subscription') => {
    return type === 'topup' ? t('Top-up') : t('Subscription')
  }

  const getStatusBadge = (record: InvoiceRecord) => {
    if (record.status === 1) {
      return (
        <Badge variant='secondary'>{t('Pending Invoice')}</Badge>
      )
    }
    if (record.status === 2) {
      return <Badge variant='default'>{t('Invoiced')}</Badge>
    }
    // status === 3
    return (
      <Badge
        variant='destructive'
        title={record.reject_reason}
      >
        {t('Rejected')}
      </Badge>
    )
  }

  const allSelected =
    invoiceableOrders.length > 0 &&
    selectedOrderIds.size === invoiceableOrders.length
  const someSelected =
    selectedOrderIds.size > 0 &&
    selectedOrderIds.size < invoiceableOrders.length

  const selectedTotal = selectedOrders.reduce((sum, o) => sum + o.money, 0)

  return (
    <>
      <Card>
        <CardHeader className='border-b pb-4'>
          <CardTitle>{t('Invoice')}</CardTitle>
        </CardHeader>

        <CardContent className='space-y-6 p-3 sm:p-5'>
          {/* Invoiceable orders section */}
          <div className='space-y-3'>
            <div className='flex items-center justify-between'>
              <h3 className='text-sm font-medium'>
                {t('Invoiceable Orders')}
              </h3>
              {selectedOrderIds.size > 0 && (
                <div className='flex items-center gap-3 text-sm'>
                  <span className='text-muted-foreground'>
                    {t('Selected:')} {selectedOrderIds.size} &nbsp;/&nbsp; $
                    {selectedTotal.toFixed(2)}
                  </span>
                  <Button
                    size='sm'
                    onClick={() => setApplyDialogOpen(true)}
                  >
                    {t('Apply for Invoice')}
                  </Button>
                </div>
              )}
              {selectedOrderIds.size === 0 && !loading && invoiceableOrders.length > 0 && (
                <Button
                  size='sm'
                  disabled
                  variant='outline'
                >
                  {t('Apply for Invoice')}
                </Button>
              )}
            </div>

            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className='w-8'>
                    {invoiceableOrders.length > 0 && (
                      <Checkbox
                        checked={someSelected ? 'mixed' : allSelected}
                        onCheckedChange={toggleAll}
                        aria-label={t('Select all')}
                      />
                    )}
                  </TableHead>
                  <TableHead>{t('Type')}</TableHead>
                  <TableHead>{t('Trade No')}</TableHead>
                  <TableHead>{t('Time')}</TableHead>
                  <TableHead className='text-right'>{t('Amount')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {loading ? (
                  <TableRow>
                    <TableCell colSpan={5} className='text-center py-8'>
                      <Loader2 className='mx-auto h-5 w-5 animate-spin text-muted-foreground' />
                    </TableCell>
                  </TableRow>
                ) : invoiceableOrders.length === 0 ? (
                  <TableRow>
                    <TableCell
                      colSpan={5}
                      className='text-center py-8 text-muted-foreground'
                    >
                      {t('No invoiceable orders')}
                    </TableCell>
                  </TableRow>
                ) : (
                  invoiceableOrders.map((order) => (
                    <TableRow key={`${order.order_type}-${order.order_id}`}>
                      <TableCell>
                        <Checkbox
                          checked={selectedOrderIds.has(`${order.order_type}-${order.order_id}`)}
                          onCheckedChange={() => toggleOrder(order)}
                          aria-label={t('Select order')}
                        />
                      </TableCell>
                      <TableCell>{getOrderTypeLabel(order.order_type)}</TableCell>
                      <TableCell>
                        <span className='font-mono text-xs'>
                          {order.trade_no}
                        </span>
                      </TableCell>
                      <TableCell>
                        {new Date(order.create_time * 1000).toLocaleString()}
                      </TableCell>
                      <TableCell className='text-right font-medium'>
                        ${order.money.toFixed(2)}
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </div>

          {/* Invoice records section */}
          <div className='space-y-3'>
            <h3 className='text-sm font-medium'>{t('My Invoice Applications')}</h3>

            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t('Invoice No')}</TableHead>
                  <TableHead>{t('Title')}</TableHead>
                  <TableHead className='text-right'>{t('Amount')}</TableHead>
                  <TableHead>{t('Status')}</TableHead>
                  <TableHead>{t('Applied At')}</TableHead>
                  <TableHead className='text-right'>{t('Actions')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {loading ? (
                  <TableRow>
                    <TableCell colSpan={6} className='text-center py-8'>
                      <Loader2 className='mx-auto h-5 w-5 animate-spin text-muted-foreground' />
                    </TableCell>
                  </TableRow>
                ) : invoiceRecords.length === 0 ? (
                  <TableRow>
                    <TableCell
                      colSpan={6}
                      className='text-center py-8 text-muted-foreground'
                    >
                      {t('No invoice applications yet')}
                    </TableCell>
                  </TableRow>
                ) : (
                  invoiceRecords.map((record) => (
                    <TableRow key={record.id}>
                      <TableCell>
                        <span className='font-mono text-xs'>
                          {record.invoice_no}
                        </span>
                      </TableCell>
                      <TableCell>{record.title_name}</TableCell>
                      <TableCell className='text-right font-medium'>
                        ${record.money.toFixed(2)}
                      </TableCell>
                      <TableCell>{getStatusBadge(record)}</TableCell>
                      <TableCell>
                        {new Date(record.create_time * 1000).toLocaleString()}
                      </TableCell>
                      <TableCell className='text-right'>
                        <div className='flex items-center justify-end gap-1'>
                          {record.status === 1 && (
                            <Button
                              size='sm'
                              variant='ghost'
                              onClick={() => handleCancel(record)}
                              disabled={cancellingId === record.id}
                              aria-label={t('Cancel')}
                            >
                              {cancellingId === record.id ? (
                                <Loader2 className='h-3.5 w-3.5 animate-spin' />
                              ) : (
                                <X className='h-3.5 w-3.5' />
                              )}
                              <span className='ml-1'>{t('Cancel')}</span>
                            </Button>
                          )}
                          {record.status === 2 && record.has_file && (
                            <Button
                              size='sm'
                              variant='ghost'
                              onClick={() => handleDownload(record)}
                              disabled={downloadingId === record.id}
                              aria-label={t('Download')}
                            >
                              {downloadingId === record.id ? (
                                <Loader2 className='h-3.5 w-3.5 animate-spin' />
                              ) : (
                                <Download className='h-3.5 w-3.5' />
                              )}
                              <span className='ml-1'>{t('Download')}</span>
                            </Button>
                          )}
                        </div>
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </div>
        </CardContent>
      </Card>

      <InvoiceApplyDialog
        open={applyDialogOpen}
        onOpenChange={setApplyDialogOpen}
        selectedOrders={selectedOrders}
        defaultTitle={defaultTitle}
        onSuccess={refresh}
      />
    </>
  )
}
