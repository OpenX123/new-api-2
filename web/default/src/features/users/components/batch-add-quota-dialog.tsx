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
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Dialog } from '@/components/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { getCurrencyDisplay, getCurrencyLabel } from '@/lib/currency'
import { formatQuota, parseQuotaFromDollars } from '@/lib/format'

import { batchAddUserQuota } from '../api'
import type { User } from '../types'

interface BatchAddQuotaDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  users: User[]
  onSuccess: () => void
}

export function BatchAddQuotaDialog(props: BatchAddQuotaDialogProps) {
  const { t } = useTranslation()
  const [amount, setAmount] = useState('300')
  const [loading, setLoading] = useState(false)
  const { meta: currencyMeta } = getCurrencyDisplay()
  const currencyLabel = getCurrencyLabel()
  const tokensOnly = currencyMeta.kind === 'tokens'
  const quotaValue = parseQuotaFromDollars(Number.parseFloat(amount) || 0)

  const handleOpenChange = (open: boolean) => {
    if (!open) setAmount('300')
    props.onOpenChange(open)
  }

  const handleConfirm = async () => {
    if (quotaValue <= 0 || props.users.length === 0) return

    setLoading(true)
    try {
      const result = await batchAddUserQuota({
        ids: props.users.map((user) => user.id),
        value: quotaValue,
      })
      if (!result.success) {
        toast.error(result.message || t('Failed to adjust quota'))
        return
      }
      if (result.data === props.users.length) {
        toast.success(t('Quota adjusted successfully'))
      } else if (result.data === 0) {
        toast.warning(t('No changes made'))
      } else {
        toast.warning(`${t('Updated')}: ${result.data}/${props.users.length}`)
      }
      handleOpenChange(false)
      props.onSuccess()
    } catch (error: unknown) {
      toast.error(
        error instanceof Error ? error.message : t('Failed to adjust quota')
      )
    } finally {
      setLoading(false)
    }
  }

  return (
    <Dialog
      open={props.open}
      onOpenChange={handleOpenChange}
      title={t('Add Quota')}
      description={`${props.users.length} ${t('selected')}`}
      contentHeight='auto'
      bodyClassName='space-y-4'
      footer={
        <>
          <Button variant='outline' onClick={() => handleOpenChange(false)}>
            {t('Cancel')}
          </Button>
          <Button onClick={handleConfirm} disabled={loading || quotaValue <= 0}>
            {loading ? t('Processing...') : t('Confirm')}
          </Button>
        </>
      }
    >
      <div className='space-y-4'>
        <div className='text-muted-foreground text-sm'>
          +{formatQuota(quotaValue)} × {props.users.length}
        </div>
        <div className='space-y-2'>
          <Label htmlFor='batch-quota-amount'>
            {t('Amount')} ({currencyLabel})
          </Label>
          <Input
            id='batch-quota-amount'
            type='number'
            step={tokensOnly ? 1 : 0.000001}
            min={tokensOnly ? 1 : 0.000001}
            value={amount}
            onChange={(event) => setAmount(event.target.value)}
            onKeyDown={(event) => {
              if (event.key === 'Enter') handleConfirm()
            }}
            autoFocus
          />
        </div>
      </div>
    </Dialog>
  )
}
