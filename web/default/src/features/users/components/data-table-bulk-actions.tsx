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
import type { Table } from '@tanstack/react-table'
import { Plus } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { DataTableBulkActions as BulkActionsToolbar } from '@/components/data-table'
import { Button } from '@/components/ui/button'

import type { User } from '../types'
import { BatchAddQuotaDialog } from './batch-add-quota-dialog'
import { useUsers } from './users-provider'

interface DataTableBulkActionsProps {
  table: Table<User>
}

export function DataTableBulkActions({ table }: DataTableBulkActionsProps) {
  const { t } = useTranslation()
  const { triggerRefresh } = useUsers()
  const [quotaDialogOpen, setQuotaDialogOpen] = useState(false)
  const selectedUsers = table
    .getFilteredSelectedRowModel()
    .rows.map((row) => row.original)

  return (
    <>
      <BulkActionsToolbar table={table} entityName='user'>
        <Button
          size='sm'
          className='h-7 gap-1.5'
          onClick={() => setQuotaDialogOpen(true)}
        >
          <Plus className='size-4' />
          <span className='hidden sm:inline'>{t('Add Quota')}</span>
        </Button>
      </BulkActionsToolbar>
      <BatchAddQuotaDialog
        open={quotaDialogOpen}
        onOpenChange={setQuotaDialogOpen}
        users={selectedUsers}
        onSuccess={() => {
          table.resetRowSelection()
          triggerRefresh()
        }}
      />
    </>
  )
}
