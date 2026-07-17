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

import { Dialog } from '@/components/dialog'
import {
  type AnnouncementItem,
  NotificationContent,
  type NotificationTab,
} from '@/components/notification-popover'
import { Button } from '@/components/ui/button'

interface HomeNoticeDialogProps {
  open: boolean
  onClose: () => void
  notice: string
  announcements: AnnouncementItem[]
  loading: boolean
}

export function HomeNoticeDialog(props: HomeNoticeDialogProps) {
  const { t } = useTranslation()
  const [activeTab, setActiveTab] = useState<NotificationTab>('notice')

  return (
    <Dialog
      open={props.open}
      onOpenChange={() => undefined}
      title={t('System Announcements')}
      description={t('Latest platform updates and notices')}
      contentClassName='sm:max-w-xl'
      contentHeight='auto'
      showCloseButton={false}
      footer={<Button onClick={props.onClose}>{t('Close')}</Button>}
    >
      <NotificationContent
        activeTab={activeTab}
        onTabChange={setActiveTab}
        notice={props.notice}
        announcements={props.announcements}
        loading={props.loading}
      />
    </Dialog>
  )
}
