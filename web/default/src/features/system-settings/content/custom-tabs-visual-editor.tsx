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
import { Plus, Search } from 'lucide-react'
import { useState, useMemo } from 'react'
import { useTranslation } from 'react-i18next'

import { StaticDataTable } from '@/components/data-table/static/static-data-table'
import { StaticRowActions } from '@/components/data-table/static/static-row-actions'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'

import { safeJsonParseWithValidation } from '../utils/json-parser'
import { isArray } from '../utils/json-validators'
import { CustomTabDialog, type CustomTabEntryData } from './custom-tab-dialog'

type CustomTabsVisualEditorProps = {
  value: string
  onChange: (value: string) => void
}

type CustomTabRow = CustomTabEntryData & { index: number }

export function CustomTabsVisualEditor({
  value,
  onChange,
}: CustomTabsVisualEditorProps) {
  const { t } = useTranslation()
  const [searchText, setSearchText] = useState('')
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editIndex, setEditIndex] = useState<number | null>(null)

  const tabs = useMemo(() => {
    const parsed = safeJsonParseWithValidation<unknown[]>(value, {
      fallback: [],
      validator: isArray,
      validatorMessage: 'Custom tabs must be a JSON array',
      context: 'custom-tabs',
    })

    return parsed
      .map((item, index) => {
        if (!item || typeof item !== 'object' || Array.isArray(item)) {
          return null
        }
        const { name, url } = item as Record<string, unknown>
        if (typeof name !== 'string' || typeof url !== 'string') {
          return null
        }
        return { name, url, index }
      })
      .filter((item): item is CustomTabRow => item !== null)
  }, [value])

  const filteredTabs = useMemo(() => {
    if (!searchText) return tabs
    const lowerSearch = searchText.toLowerCase()
    return tabs.filter(
      (tab) =>
        tab.name.toLowerCase().includes(lowerSearch) ||
        tab.url.toLowerCase().includes(lowerSearch)
    )
  }, [searchText, tabs])

  const handleSave = (data: CustomTabEntryData) => {
    const updated = tabs.map(({ name, url }) => ({ name, url }))
    const position =
      editIndex === null ? -1 : tabs.findIndex((tab) => tab.index === editIndex)

    if (position === -1) {
      updated.push(data)
    } else {
      updated[position] = data
    }

    onChange(JSON.stringify(updated, null, 2))
  }

  const handleDelete = (index: number) => {
    const updated = tabs
      .filter((tab) => tab.index !== index)
      .map(({ name, url }) => ({ name, url }))

    onChange(JSON.stringify(updated, null, 2))
  }

  const handleEdit = (tab: CustomTabRow) => {
    setEditIndex(tab.index)
    setDialogOpen(true)
  }

  const handleAdd = () => {
    setEditIndex(null)
    setDialogOpen(true)
  }

  const editData = useMemo(() => {
    if (editIndex === null) return null
    const target = tabs.find((tab) => tab.index === editIndex)
    return target ? { name: target.name, url: target.url } : null
  }, [editIndex, tabs])

  return (
    <div className='space-y-4'>
      <div className='flex items-center gap-4'>
        <div className='relative flex-1'>
          <Search className='text-muted-foreground absolute top-2.5 left-2.5 h-4 w-4' />
          <Input
            placeholder={t('Search custom tabs...')}
            value={searchText}
            onChange={(e) => setSearchText(e.target.value)}
            className='pl-9'
          />
        </div>
        <Button onClick={handleAdd}>
          <Plus className='mr-2 h-4 w-4' />
          {t('Add custom tab')}
        </Button>
      </div>

      <StaticDataTable
        data={filteredTabs}
        getRowKey={(tab) => String(tab.index)}
        emptyContent={
          searchText
            ? t('No custom tabs match your search')
            : t(
                'No custom tabs configured. Click "Add custom tab" to get started.'
              )
        }
        columns={[
          {
            id: 'name',
            header: t('Tab Name'),
            cellClassName: 'font-medium',
            cell: (tab) => tab.name,
          },
          {
            id: 'url',
            header: t('URL'),
            cellClassName: 'max-w-md truncate font-mono text-sm',
            cell: (tab) => tab.url,
          },
          {
            id: 'actions',
            header: t('Actions'),
            className: 'text-right',
            cellClassName: 'text-right',
            cell: (tab) => (
              <StaticRowActions
                editLabel={t('Edit')}
                deleteLabel={t('Delete')}
                menuLabel={t('Open menu')}
                onEdit={() => handleEdit(tab)}
                onDelete={() => handleDelete(tab.index)}
              />
            ),
          },
        ]}
      />

      <CustomTabDialog
        open={dialogOpen}
        onOpenChange={setDialogOpen}
        onSave={handleSave}
        editData={editData}
      />
    </div>
  )
}
