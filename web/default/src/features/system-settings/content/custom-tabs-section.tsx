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
import { useEffect, useRef, useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import * as z from 'zod'

import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Textarea } from '@/components/ui/textarea'

import { SettingsForm } from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useUpdateOption } from '../hooks/use-update-option'
import { CustomTabsVisualEditor } from './custom-tabs-visual-editor'
import { formatJsonForEditor, normalizeJsonString } from './utils'

const createCustomTabsSchema = (t: (key: string) => string) =>
  z.object({
    CustomTabs: z.string().superRefine((value, ctx) => {
      let parsed: unknown
      try {
        parsed = JSON.parse(value || '[]')
      } catch {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          message: t('Invalid JSON string.'),
        })
        return
      }

      if (!Array.isArray(parsed)) {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          message: t('Expected a JSON array.'),
        })
        return
      }

      for (const item of parsed) {
        if (item === null || typeof item !== 'object' || Array.isArray(item)) {
          ctx.addIssue({
            code: z.ZodIssueCode.custom,
            message: t('Each item must be an object with "name" and "url".'),
          })
          return
        }
        const { name, url } = item as Record<string, unknown>
        if (typeof name !== 'string' || !name.trim()) {
          ctx.addIssue({
            code: z.ZodIssueCode.custom,
            message: t('Each item must have a non-empty "name".'),
          })
          return
        }
        if (typeof url !== 'string' || !/^https?:\/\//i.test(url.trim())) {
          ctx.addIssue({
            code: z.ZodIssueCode.custom,
            message: t('Each "url" must start with http:// or https://'),
          })
          return
        }
      }
    }),
  })

type CustomTabsFormValues = z.infer<ReturnType<typeof createCustomTabsSchema>>

type CustomTabsSectionProps = {
  defaultValue: string
}

export function CustomTabsSection({ defaultValue }: CustomTabsSectionProps) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const [editMode, setEditMode] = useState<'visual' | 'json'>('visual')

  const customTabsSchema = createCustomTabsSchema(t)
  const form = useForm<CustomTabsFormValues>({
    resolver: zodResolver(customTabsSchema),
    mode: 'onChange',
    defaultValues: {
      CustomTabs: formatJsonForEditor(defaultValue, '[]'),
    },
  })

  const initialNormalizedRef = useRef(normalizeJsonString(defaultValue, '[]'))

  useEffect(() => {
    form.reset({ CustomTabs: formatJsonForEditor(defaultValue, '[]') })
    initialNormalizedRef.current = normalizeJsonString(defaultValue, '[]')
  }, [defaultValue, form])

  const onSubmit = async (values: CustomTabsFormValues) => {
    const normalized = normalizeJsonString(values.CustomTabs, '[]')
    if (normalized === initialNormalizedRef.current) {
      return
    }

    await updateOption.mutateAsync({
      key: 'CustomTabs',
      value: normalized,
    })
  }

  return (
    <SettingsSection title={t('Custom Tabs')}>
      <Form {...form}>
        {/* eslint-disable-next-line react-hooks/refs */}
        <SettingsForm onSubmit={form.handleSubmit(onSubmit)}>
          <SettingsPageFormActions
            onSave={form.handleSubmit(onSubmit)}
            isSaving={updateOption.isPending}
            saveLabel='Save custom tabs'
          />
          <Tabs
            value={editMode}
            onValueChange={(value) => setEditMode(value as 'visual' | 'json')}
          >
            <TabsList className='grid w-full grid-cols-2'>
              <TabsTrigger value='visual'>{t('Visual')}</TabsTrigger>
              <TabsTrigger value='json'>{t('JSON')}</TabsTrigger>
            </TabsList>

            <TabsContent value='visual' className='mt-6'>
              <FormField
                control={form.control}
                name='CustomTabs'
                render={({ field }) => (
                  <FormItem>
                    <FormControl>
                      <CustomTabsVisualEditor
                        value={field.value}
                        onChange={field.onChange}
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </TabsContent>

            <TabsContent value='json' className='mt-6'>
              <FormField
                control={form.control}
                name='CustomTabs'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Custom tabs configuration JSON')}</FormLabel>
                    <FormControl>
                      <Textarea
                        rows={12}
                        placeholder={t(
                          '[{"name":"Skill Hub","url":"https://hub.example.com/?key={key}"}]'
                        )}
                        {...field}
                      />
                    </FormControl>
                    <FormDescription>
                      {t(
                        'Array of pages embedded into the console sidebar as iframes. Each item has a name and an https URL, which may contain {key} and {address} placeholders.'
                      )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </TabsContent>
          </Tabs>
        </SettingsForm>
      </Form>
    </SettingsSection>
  )
}
