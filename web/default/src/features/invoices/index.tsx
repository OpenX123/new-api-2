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
import { useCallback, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { SectionPageLayout } from '@/components/layout'
import { getInvoices } from './api'
import { InvoiceDetailDrawer } from './components/invoice-detail-drawer'
import type { AdminInvoice } from './types'

const PAGE_SIZE = 20

function getStatusLabel(status: number, t: (key: string) => string) {
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

function getStatusVariant(
  status: number
): 'default' | 'secondary' | 'destructive' | 'outline' {
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

export function Invoices() {
  const { t } = useTranslation()
  const [items, setItems] = useState<AdminInvoice[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [keyword, setKeyword] = useState('')
  const [status, setStatus] = useState<number>(0)
  const [loading, setLoading] = useState(false)

  const [selectedId, setSelectedId] = useState<number | null>(null)
  const [drawerOpen, setDrawerOpen] = useState(false)

  const fetchInvoices = useCallback(
    async (p: number, kw: string, st: number) => {
      setLoading(true)
      try {
        const res = await getInvoices({
          p,
          page_size: PAGE_SIZE,
          status: st,
          keyword: kw,
        })
        if (res.success && res.data) {
          setItems(res.data.items ?? [])
          setTotal(res.data.total ?? 0)
        } else {
          setItems([])
          setTotal(0)
        }
      } catch {
        setItems([])
        setTotal(0)
      } finally {
        setLoading(false)
      }
    },
    []
  )

  useEffect(() => {
    void fetchInvoices(page, keyword, status)
  }, [fetchInvoices, page, keyword, status])

  const handleSearch = (value: string) => {
    setKeyword(value)
    setPage(1)
  }

  const handleStatusChange = (value: string) => {
    setStatus(Number(value))
    setPage(1)
  }

  const handleRowClick = (id: number) => {
    setSelectedId(id)
    setDrawerOpen(true)
  }

  const handleChanged = () => {
    void fetchInvoices(page, keyword, status)
  }

  const totalPages = Math.ceil(total / PAGE_SIZE)

  return (
    <>
      <SectionPageLayout fixedContent>
        <SectionPageLayout.Title>
          {t('Invoice Management')}
        </SectionPageLayout.Title>
        <SectionPageLayout.Actions>
          <div className='flex items-center gap-2'>
            <Select
              value={String(status)}
              onValueChange={handleStatusChange}
            >
              <SelectTrigger className='w-36'>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value='0'>{t('All Status')}</SelectItem>
                <SelectItem value='1'>{t('Pending Invoice')}</SelectItem>
                <SelectItem value='2'>{t('Invoiced')}</SelectItem>
                <SelectItem value='3'>{t('Rejected')}</SelectItem>
              </SelectContent>
            </Select>
            <Input
              className='w-48'
              placeholder={t('Search keyword...')}
              value={keyword}
              onChange={(e) => handleSearch(e.target.value)}
            />
          </div>
        </SectionPageLayout.Actions>
        <SectionPageLayout.Content>
          <div className='space-y-4'>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t('Invoice No')}</TableHead>
                  <TableHead>{t('User')}</TableHead>
                  <TableHead>{t('Title Name')}</TableHead>
                  <TableHead className='text-right'>
                    {t('Amount')}
                  </TableHead>
                  <TableHead>{t('Status')}</TableHead>
                  <TableHead>{t('Create Time')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {loading && (
                  <TableRow>
                    <TableCell
                      colSpan={6}
                      className='text-muted-foreground text-center'
                    >
                      {t('Loading...')}
                    </TableCell>
                  </TableRow>
                )}
                {!loading && items.length === 0 && (
                  <TableRow>
                    <TableCell
                      colSpan={6}
                      className='text-muted-foreground text-center'
                    >
                      {t('No data')}
                    </TableCell>
                  </TableRow>
                )}
                {!loading &&
                  items.map((item) => (
                    <TableRow
                      key={item.id}
                      className='cursor-pointer'
                      onClick={() => handleRowClick(item.id)}
                    >
                      <TableCell className='font-mono text-sm'>
                        {item.invoice_no}
                      </TableCell>
                      <TableCell>{item.username}</TableCell>
                      <TableCell>{item.title_name}</TableCell>
                      <TableCell className='text-right'>
                        ${item.money.toFixed(2)}
                      </TableCell>
                      <TableCell>
                        <Badge variant={getStatusVariant(item.status)}>
                          {getStatusLabel(item.status, t)}
                        </Badge>
                      </TableCell>
                      <TableCell>
                        {new Date(
                          item.create_time * 1000
                        ).toLocaleString()}
                      </TableCell>
                    </TableRow>
                  ))}
              </TableBody>
            </Table>

            {/* Pagination */}
            <div className='flex items-center justify-between text-sm'>
              <span className='text-muted-foreground'>
                {t('Total')}: {total}
              </span>
              <div className='flex items-center gap-2'>
                <Button
                  variant='outline'
                  size='sm'
                  disabled={page <= 1}
                  onClick={() => setPage((p) => p - 1)}
                >
                  {t('Previous')}
                </Button>
                <span>
                  {page} / {totalPages || 1}
                </span>
                <Button
                  variant='outline'
                  size='sm'
                  disabled={page >= totalPages}
                  onClick={() => setPage((p) => p + 1)}
                >
                  {t('Next')}
                </Button>
              </div>
            </div>
          </div>
        </SectionPageLayout.Content>
      </SectionPageLayout>

      <InvoiceDetailDrawer
        invoiceId={selectedId}
        open={drawerOpen}
        onOpenChange={setDrawerOpen}
        onChanged={handleChanged}
      />
    </>
  )
}
