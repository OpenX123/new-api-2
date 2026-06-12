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
import { useState, useEffect } from 'react'
import { Loader2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Textarea } from '@/components/ui/textarea'
import { Dialog } from '@/components/dialog'
import { createInvoice, isApiSuccess } from '../../api'
import type { InvoiceableOrder, InvoiceTitle } from '../../types'

interface InvoiceApplyDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  selectedOrders: InvoiceableOrder[]
  defaultTitle: InvoiceTitle | null
  onSuccess: () => void
}

export function InvoiceApplyDialog({
  open,
  onOpenChange,
  selectedOrders,
  defaultTitle,
  onSuccess,
}: InvoiceApplyDialogProps) {
  const { t } = useTranslation()
  const [titleType, setTitleType] = useState<'1' | '2'>('1')
  const [titleName, setTitleName] = useState('')
  const [taxNo, setTaxNo] = useState('')
  const [email, setEmail] = useState('')
  const [remark, setRemark] = useState('')
  const [saveAsDefault, setSaveAsDefault] = useState(true)
  const [submitting, setSubmitting] = useState(false)
  const [errors, setErrors] = useState<Record<string, string>>({})

  // Prefill from defaultTitle when dialog opens
  useEffect(() => {
    if (open && defaultTitle) {
      setTitleType(String(defaultTitle.title_type) as '1' | '2')
      setTitleName(defaultTitle.title_name)
      setTaxNo(defaultTitle.tax_no)
      setEmail(defaultTitle.email)
      setRemark('')
      setSaveAsDefault(false)
      setErrors({})
    } else if (open) {
      setTitleType('1')
      setTitleName('')
      setTaxNo('')
      setEmail('')
      setRemark('')
      setSaveAsDefault(true)
      setErrors({})
    }
  }, [open, defaultTitle])

  const totalAmount = selectedOrders.reduce((sum, o) => sum + o.money, 0)

  const validate = (): boolean => {
    const newErrors: Record<string, string> = {}
    if (!titleName.trim()) {
      newErrors.titleName = t('Invoice title is required')
    }
    if (!email.trim()) {
      newErrors.email = t('Email is required')
    } else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)) {
      newErrors.email = t('Invalid email address')
    }
    if (titleType === '2' && !taxNo.trim()) {
      newErrors.taxNo = t('Tax number is required for company invoices')
    }
    setErrors(newErrors)
    return Object.keys(newErrors).length === 0
  }

  const handleSubmit = async () => {
    if (!validate()) return

    setSubmitting(true)
    try {
      const res = await createInvoice({
        order_keys: selectedOrders.map((o) => ({
          type: o.order_type,
          id: o.order_id,
        })),
        title_type: Number(titleType) as 1 | 2,
        title_name: titleName.trim(),
        tax_no: taxNo.trim(),
        email: email.trim(),
        remark: remark.trim(),
        save_as_default: saveAsDefault,
      })
      if (isApiSuccess(res)) {
        toast.success(t('Invoice application submitted successfully'))
        onOpenChange(false)
        onSuccess()
      } else {
        toast.error(res.message || t('Failed to submit invoice application'))
      }
    } catch {
      toast.error(t('Failed to submit invoice application'))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      title={t('Apply for Invoice')}
      description={t('Fill in your invoice details below')}
      contentClassName='max-sm:w-[calc(100vw-1.5rem)] sm:max-w-lg'
      titleClassName='text-xl font-semibold'
      footerClassName='grid grid-cols-2 gap-2 sm:flex'
      contentHeight='auto'
      bodyClassName='space-y-4'
      showCloseButton
      footer={
        <>
          <Button
            variant='outline'
            onClick={() => onOpenChange(false)}
            disabled={submitting}
          >
            {t('Cancel')}
          </Button>
          <Button onClick={handleSubmit} disabled={submitting}>
            {submitting && <Loader2 className='mr-2 h-4 w-4 animate-spin' />}
            {t('Submit')}
          </Button>
        </>
      }
    >
      <div className='space-y-4 py-2'>
        {/* Order summary */}
        <div className='bg-muted/50 rounded-lg px-3 py-2 text-sm'>
          <span className='text-muted-foreground'>
            {t('Selected orders:')} {selectedOrders.length}
          </span>
          <span className='ml-3 font-medium'>
            {t('Total:')} ${totalAmount.toFixed(2)}
          </span>
        </div>

        {/* Invoice title type */}
        <div className='space-y-2'>
          <Label className='text-muted-foreground text-xs font-medium tracking-wider uppercase'>
            {t('Invoice Type')}
          </Label>
          <Select
            value={titleType}
            onValueChange={(v) => {
              setTitleType(v as '1' | '2')
              setErrors((prev) => ({ ...prev, taxNo: '' }))
            }}
          >
            <SelectTrigger className='w-full'>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value='1'>{t('Personal')}</SelectItem>
              <SelectItem value='2'>{t('Company')}</SelectItem>
            </SelectContent>
          </Select>
        </div>

        {/* Title name */}
        <div className='space-y-2'>
          <Label
            htmlFor='invoice-title-name'
            className='text-muted-foreground text-xs font-medium tracking-wider uppercase'
          >
            {titleType === '2' ? t('Company Name') : t('Invoice Title')}
          </Label>
          <Input
            id='invoice-title-name'
            value={titleName}
            onChange={(e) => {
              setTitleName(e.target.value)
              setErrors((prev) => ({ ...prev, titleName: '' }))
            }}
            aria-invalid={!!errors.titleName}
            placeholder={
              titleType === '2'
                ? t('Enter company name')
                : t('Enter your name')
            }
          />
          {errors.titleName && (
            <p className='text-destructive text-xs'>{errors.titleName}</p>
          )}
        </div>

        {/* Tax number — only for company */}
        {titleType === '2' && (
          <div className='space-y-2'>
            <Label
              htmlFor='invoice-tax-no'
              className='text-muted-foreground text-xs font-medium tracking-wider uppercase'
            >
              {t('Tax Number')}
            </Label>
            <Input
              id='invoice-tax-no'
              value={taxNo}
              onChange={(e) => {
                setTaxNo(e.target.value)
                setErrors((prev) => ({ ...prev, taxNo: '' }))
              }}
              aria-invalid={!!errors.taxNo}
              placeholder={t('Enter tax number')}
            />
            {errors.taxNo && (
              <p className='text-destructive text-xs'>{errors.taxNo}</p>
            )}
          </div>
        )}

        {/* Email */}
        <div className='space-y-2'>
          <Label
            htmlFor='invoice-email'
            className='text-muted-foreground text-xs font-medium tracking-wider uppercase'
          >
            {t('Recipient Email')}
          </Label>
          <Input
            id='invoice-email'
            type='email'
            value={email}
            onChange={(e) => {
              setEmail(e.target.value)
              setErrors((prev) => ({ ...prev, email: '' }))
            }}
            aria-invalid={!!errors.email}
            placeholder={t('Enter email address')}
          />
          {errors.email && (
            <p className='text-destructive text-xs'>{errors.email}</p>
          )}
        </div>

        {/* Remark */}
        <div className='space-y-2'>
          <Label
            htmlFor='invoice-remark'
            className='text-muted-foreground text-xs font-medium tracking-wider uppercase'
          >
            {t('Remark')} ({t('Optional')})
          </Label>
          <Textarea
            id='invoice-remark'
            value={remark}
            onChange={(e) => setRemark(e.target.value)}
            placeholder={t('Any additional notes')}
            className='resize-none'
            rows={2}
          />
        </div>

        {/* Save as default */}
        <div className='flex items-center gap-2'>
          <Checkbox
            id='invoice-save-default'
            checked={saveAsDefault}
            onCheckedChange={(checked) => setSaveAsDefault(checked === true)}
          />
          <Label
            htmlFor='invoice-save-default'
            className='cursor-pointer text-sm'
          >
            {t('Save as default invoice title')}
          </Label>
        </div>
      </div>
    </Dialog>
  )
}
