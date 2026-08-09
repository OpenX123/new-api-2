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
import { Link, createFileRoute, redirect } from '@tanstack/react-router'
import { Loader2, MessageCircleWarning } from 'lucide-react'
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'

import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { useActiveChatKey } from '@/features/chat/hooks/use-active-chat-key'
import {
  chatLinkRequiresApiKey,
  resolveChatUrl,
} from '@/features/chat/lib/chat-links'
import { useCustomTabs } from '@/features/custom-tabs/hooks/use-custom-tabs'

export const Route = createFileRoute('/_authenticated/embed/$tabId')({
  loader: async ({ params }) => {
    if (!Number.isInteger(Number(params.tabId))) {
      throw redirect({ to: '/dashboard' })
    }
  },
  component: EmbedRouteComponent,
})

function EmbedRouteComponent() {
  const { t } = useTranslation()
  const { tabId } = Route.useParams()
  const { customTabs, serverAddress } = useCustomTabs()

  const tab = useMemo(() => {
    const index = Number(tabId)
    if (!Number.isInteger(index)) return undefined
    return customTabs[index]
  }, [customTabs, tabId])

  const requiresActiveKey = useMemo(() => {
    if (!tab) return false
    return chatLinkRequiresApiKey(tab.url)
  }, [tab])

  const {
    data: activeKey,
    isPending,
    isError,
    error,
  } = useActiveChatKey(Boolean(tab && requiresActiveKey))

  const iframeSrc = useMemo(() => {
    if (!tab) return ''
    if (requiresActiveKey && !activeKey) return ''
    return resolveChatUrl({
      template: tab.url,
      apiKey: requiresActiveKey ? activeKey : undefined,
      serverAddress,
    })
  }, [activeKey, requiresActiveKey, serverAddress, tab])

  if (!tab) {
    return (
      <div className='flex h-full flex-col items-center justify-center gap-4 p-6 text-center'>
        <MessageCircleWarning className='text-muted-foreground h-12 w-12' />
        <div className='space-y-1'>
          <h2 className='text-lg font-semibold'>{t('Custom tab not found')}</h2>
          <p className='text-muted-foreground'>
            {t('The requested custom tab does not exist or has been removed.')}
          </p>
        </div>
        <Button variant='outline' render={<Link to='/dashboard' />}>
          {t('Return to dashboard')}
        </Button>
      </div>
    )
  }

  if (requiresActiveKey && isPending) {
    return (
      <div className='flex h-full flex-col items-center justify-center gap-4'>
        <Loader2 className='text-muted-foreground h-8 w-8 animate-spin' />
        <p className='text-muted-foreground text-sm'>
          {t('Preparing this page…')}
        </p>
      </div>
    )
  }

  if (requiresActiveKey && (isError || !activeKey || !iframeSrc)) {
    const message =
      error instanceof Error
        ? error.message
        : 'Unable to open this page. Please check your API keys.'
    return (
      <div className='flex h-full flex-col items-center justify-center p-6'>
        <Alert variant='destructive' className='max-w-xl'>
          <AlertTitle>{t('Unable to open custom tab')}</AlertTitle>
          <AlertDescription>{message}</AlertDescription>
        </Alert>
      </div>
    )
  }

  if (!iframeSrc) {
    return (
      <div className='flex h-full flex-col items-center justify-center p-6'>
        <Alert variant='destructive' className='max-w-xl'>
          <AlertTitle>{t('Unable to open custom tab')}</AlertTitle>
          <AlertDescription>
            {t(
              'Unable to resolve this page URL. Please contact your administrator.'
            )}
          </AlertDescription>
        </Alert>
      </div>
    )
  }

  return (
    <iframe
      src={iframeSrc}
      key={iframeSrc}
      className='h-full w-full border-0'
      allow='camera; microphone; clipboard-write'
      title={`Custom tab: ${tab.name}`}
    />
  )
}
