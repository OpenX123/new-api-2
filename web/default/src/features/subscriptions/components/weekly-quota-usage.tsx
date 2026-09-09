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
import { useTranslation } from 'react-i18next'

import { Progress } from '@/components/ui/progress'
import { formatQuota } from '@/lib/format'

import type { UserSubscription } from '../types'

export function WeeklyQuotaUsage(props: { subscription: UserSubscription }) {
  const { t } = useTranslation()
  const sub = props.subscription
  const total = sub.weekly_amount ?? 0
  if (total <= 0) return null

  const now = Math.floor(Date.now() / 1000)
  const active = sub.status === 'active' && sub.end_time > now
  const reset = sub.weekly_reset_time ?? 0
  const overdue = active && reset > 0 && reset <= now
  const used = overdue ? 0 : (sub.weekly_used ?? 0)
  const nextReset = overdue
    ? reset + (Math.floor((now - reset) / 604800) + 1) * 604800
    : reset
  const percent = Math.min(100, Math.max(0, (used / total) * 100))

  return (
    <div className='mt-2 min-w-0 space-y-1 text-sm'>
      <div className='flex flex-wrap items-center justify-between gap-x-3 gap-y-1'>
        <span>{t('Weekly Quota')}</span>
        <span>
          {formatQuota(used)} / {formatQuota(total)} ({Math.round(percent)}%)
        </span>
      </div>
      <Progress value={percent} className='h-1.5' />
      {active && nextReset > 0 && nextReset < sub.end_time && (
        <div className='text-muted-foreground text-xs'>
          {t('Next reset')}: {new Date(nextReset * 1000).toLocaleString()}
        </div>
      )}
    </div>
  )
}
