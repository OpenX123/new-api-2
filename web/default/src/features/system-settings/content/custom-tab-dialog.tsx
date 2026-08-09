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
import { zodResolver } from '@hookform/resolvers/zod'
import { useEffect } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import * as z from 'zod'

import { Dialog } from '@/components/dialog'
import { Button } from '@/components/ui/button'
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'

const createCustomTabDialogSchema = (t: (key: string) => string) =>
  z.object({
    name: z.string().min(1, t('Tab name is required')),
    url: z
      .string()
      .min(1, t('URL is required'))
      .refine((value) => /^https?:\/\//i.test(value.trim()), {
        message: t('URL must start with http:// or https://'),
      }),
  })

type CustomTabDialogFormValues = z.infer<
  ReturnType<typeof createCustomTabDialogSchema>
>

const CUSTOM_TAB_DIALOG_FORM_ID = 'custom-tab-dialog-form'

export type CustomTabEntryData = {
  name: string
  url: string
}

type CustomTabDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  onSave: (data: CustomTabEntryData) => void
  editData?: CustomTabEntryData | null
}

export function CustomTabDialog({
  open,
  onOpenChange,
  onSave,
  editData,
}: CustomTabDialogProps) {
  const { t } = useTranslation()
  const isEditMode = !!editData
  const customTabDialogSchema = createCustomTabDialogSchema(t)

  const form = useForm<CustomTabDialogFormValues>({
    resolver: zodResolver(customTabDialogSchema),
    defaultValues: {
      name: '',
      url: '',
    },
  })

  useEffect(() => {
    if (editData) {
      form.reset(editData)
    } else {
      form.reset({
        name: '',
        url: '',
      })
    }
  }, [editData, form, open])

  const handleSubmit = (values: CustomTabDialogFormValues) => {
    onSave({ name: values.name.trim(), url: values.url.trim() })
    form.reset()
    onOpenChange(false)
  }

  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      title={isEditMode ? t('Edit custom tab') : t('Add custom tab')}
      description={t(
        'Embed an external page into the console sidebar as an iframe.'
      )}
      contentClassName='sm:max-w-[500px]'
      contentHeight='auto'
      bodyClassName='space-y-4'
      footer={
        <>
          <Button
            type='button'
            variant='outline'
            onClick={() => onOpenChange(false)}
          >
            {t('Cancel')}
          </Button>
          <Button type='submit' form={CUSTOM_TAB_DIALOG_FORM_ID}>
            {isEditMode ? t('Update') : t('Add')}
          </Button>
        </>
      }
    >
      <Form {...form}>
        <form
          id={CUSTOM_TAB_DIALOG_FORM_ID}
          onSubmit={form.handleSubmit(handleSubmit)}
          className='space-y-4'
        >
          <FormField
            control={form.control}
            name='name'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Tab Name')}</FormLabel>
                <FormControl>
                  <Input placeholder={t('Please enter tab name')} {...field} />
                </FormControl>
                <FormDescription>
                  {t('Display name shown in the sidebar.')}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name='url'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('URL')}</FormLabel>
                <FormControl>
                  <Input placeholder={t('Please enter the URL')} {...field} />
                </FormControl>
                <FormDescription>
                  {t(
                    'Page URL to embed. Supports {key} for the user API key and {address} for the site address. The target site must allow being embedded in an iframe.'
                  )}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />
        </form>
      </Form>
    </Dialog>
  )
}
