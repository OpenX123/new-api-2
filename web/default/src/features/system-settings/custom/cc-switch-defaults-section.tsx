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
import { ChevronRight } from 'lucide-react'
import { useEffect } from 'react'
import { useForm, type Control } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import * as z from 'zod'

import { Button } from '@/components/ui/button'
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '@/components/ui/collapsible'
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'

import {
  SettingsForm,
  SettingsFormGrid,
} from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useUpdateOption } from '../hooks/use-update-option'
import type { CustomSettings } from '../types'

const ccSwitchDefaultsSchema = z.object({
  cc_switch_setting: z.object({
    claude_name: z.string().trim().min(1),
    claude_model: z.string(),
    claude_haiku_model: z.string(),
    claude_sonnet_model: z.string(),
    claude_opus_model: z.string(),
    codex_name: z.string().trim().min(1),
    codex_model: z.string(),
    gemini_name: z.string().trim().min(1),
    gemini_model: z.string(),
  }),
})

type CCSwitchFormValues = z.infer<typeof ccSwitchDefaultsSchema>

function toFormValues(settings: CustomSettings): CCSwitchFormValues {
  return {
    cc_switch_setting: {
      claude_name: settings['cc_switch_setting.claude_name'],
      claude_model: settings['cc_switch_setting.claude_model'],
      claude_haiku_model:
        settings['cc_switch_setting.claude_haiku_model'],
      claude_sonnet_model:
        settings['cc_switch_setting.claude_sonnet_model'],
      claude_opus_model: settings['cc_switch_setting.claude_opus_model'],
      codex_name: settings['cc_switch_setting.codex_name'],
      codex_model: settings['cc_switch_setting.codex_model'],
      gemini_name: settings['cc_switch_setting.gemini_name'],
      gemini_model: settings['cc_switch_setting.gemini_model'],
    },
  }
}

type CCSwitchFieldProps = {
  control: Control<CCSwitchFormValues>
  name: keyof CustomSettings
  label: string
  placeholder?: string
}

function CCSwitchField(props: CCSwitchFieldProps) {
  return (
    <FormField
      control={props.control}
      name={props.name}
      render={({ field }) => (
        <FormItem>
          <FormLabel>{props.label}</FormLabel>
          <FormControl>
            <Input placeholder={props.placeholder} {...field} />
          </FormControl>
          <FormMessage />
        </FormItem>
      )}
    />
  )
}

type CCSwitchDefaultsSectionProps = {
  defaultValues: CustomSettings
}

export function CCSwitchDefaultsSection(props: CCSwitchDefaultsSectionProps) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const form = useForm<CCSwitchFormValues>({
    resolver: zodResolver(ccSwitchDefaultsSchema),
    mode: 'onChange',
    defaultValues: toFormValues(props.defaultValues),
  })

  useEffect(() => {
    form.reset(toFormValues(props.defaultValues))
  }, [form, props.defaultValues])

  const onSubmit = async (values: CCSwitchFormValues) => {
    for (const [field, value] of Object.entries(values.cc_switch_setting)) {
      const key = `cc_switch_setting.${field}` as keyof CustomSettings
      if (value !== props.defaultValues[key]) {
        await updateOption.mutateAsync({ key, value })
      }
    }
  }

  return (
    <SettingsSection title={t('CC Switch Defaults')}>
      <Form {...form}>
        <SettingsForm onSubmit={form.handleSubmit(onSubmit)}>
          <SettingsPageFormActions
            onSave={form.handleSubmit(onSubmit)}
            onReset={() => form.reset(toFormValues(props.defaultValues))}
            isSaving={updateOption.isPending}
            isResetDisabled={!form.formState.isDirty}
            saveLabel='Save custom configuration'
          />

          <p className='text-muted-foreground text-sm'>
            {t(
              'These defaults are filled in when users import an API key to CC Switch.'
            )}
          </p>

          <Tabs defaultValue='claude'>
            <TabsList className='grid w-full grid-cols-3'>
              <TabsTrigger value='claude'>Claude</TabsTrigger>
              <TabsTrigger value='codex'>Codex</TabsTrigger>
              <TabsTrigger value='gemini'>Gemini</TabsTrigger>
            </TabsList>

            <TabsContent value='claude' className='mt-6'>
              <SettingsFormGrid>
                <CCSwitchField
                  control={form.control}
                  name='cc_switch_setting.claude_name'
                  label={t('Name')}
                  placeholder='My Claude'
                />
                <CCSwitchField
                  control={form.control}
                  name='cc_switch_setting.claude_model'
                  label={t('Primary Model')}
                  placeholder={t('Enter model name')}
                />

                <Collapsible data-settings-form-span='full'>
                  <CollapsibleTrigger
                    render={
                      <Button
                        type='button'
                        variant='outline'
                        className='group/advanced-trigger w-full justify-between'
                      />
                    }
                  >
                    {t('Advanced model mappings')}
                    <ChevronRight className='size-4 transition-transform group-data-[panel-open]/advanced-trigger:rotate-90' />
                  </CollapsibleTrigger>
                  <CollapsibleContent className='pt-6'>
                    <SettingsFormGrid>
                      <CCSwitchField
                        control={form.control}
                        name='cc_switch_setting.claude_haiku_model'
                        label={t('Haiku Model')}
                        placeholder={t('Enter model name')}
                      />
                      <CCSwitchField
                        control={form.control}
                        name='cc_switch_setting.claude_sonnet_model'
                        label={t('Sonnet Model')}
                        placeholder={t('Enter model name')}
                      />
                      <CCSwitchField
                        control={form.control}
                        name='cc_switch_setting.claude_opus_model'
                        label={t('Opus Model')}
                        placeholder={t('Enter model name')}
                      />
                    </SettingsFormGrid>
                  </CollapsibleContent>
                </Collapsible>
              </SettingsFormGrid>
            </TabsContent>

            <TabsContent value='codex' className='mt-6'>
              <SettingsFormGrid>
                <CCSwitchField
                  control={form.control}
                  name='cc_switch_setting.codex_name'
                  label={t('Name')}
                  placeholder='My Codex'
                />
                <CCSwitchField
                  control={form.control}
                  name='cc_switch_setting.codex_model'
                  label={t('Primary Model')}
                  placeholder={t('Enter model name')}
                />
              </SettingsFormGrid>
            </TabsContent>

            <TabsContent value='gemini' className='mt-6'>
              <SettingsFormGrid>
                <CCSwitchField
                  control={form.control}
                  name='cc_switch_setting.gemini_name'
                  label={t('Name')}
                  placeholder='My Gemini'
                />
                <CCSwitchField
                  control={form.control}
                  name='cc_switch_setting.gemini_model'
                  label={t('Primary Model')}
                  placeholder={t('Enter model name')}
                />
              </SettingsFormGrid>
            </TabsContent>
          </Tabs>
        </SettingsForm>
      </Form>
    </SettingsSection>
  )
}
