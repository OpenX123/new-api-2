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
import { Link, useLocation } from '@tanstack/react-router'
import { ChevronRight } from 'lucide-react'

import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '@/components/ui/collapsible'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import {
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarMenuSub,
  SidebarMenuSubButton,
  SidebarMenuSubItem,
  useSidebar,
} from '@/components/ui/sidebar'
import { useCustomTabs } from '@/features/custom-tabs/hooks/use-custom-tabs'

import { normalizeHref } from '../lib/url-utils'
import type { NavCustomTabs } from '../types'

/**
 * Dynamic custom tabs navigation item
 *
 * Renders nothing when the operator has not configured any custom tab, so the
 * enclosing nav group stays visually unchanged on default deployments.
 */
export function CustomTabsItem({ item }: { item: NavCustomTabs }) {
  const { customTabs } = useCustomTabs()
  const { state, isMobile, setOpenMobile } = useSidebar()
  const href = useLocation({ select: (location) => location.href })

  if (customTabs.length === 0) {
    return null
  }

  const normalizedHref = normalizeHref(href)

  if (state === 'collapsed' && !isMobile) {
    return (
      <SidebarMenuItem>
        <DropdownMenu>
          <DropdownMenuTrigger
            render={<SidebarMenuButton tooltip={item.title} />}
          >
            {item.icon && <item.icon className='h-4 w-4 shrink-0' />}
            <span className='min-w-0 flex-1 truncate'>{item.title}</span>
            <ChevronRight className='ms-auto h-4 w-4 shrink-0 opacity-70' />
          </DropdownMenuTrigger>
          <DropdownMenuContent align='start'>
            {customTabs.map((tab) => (
              <DropdownMenuItem
                key={tab.id}
                render={<Link to='/embed/$tabId' params={{ tabId: tab.id }} />}
              >
                {tab.name}
              </DropdownMenuItem>
            ))}
          </DropdownMenuContent>
        </DropdownMenu>
      </SidebarMenuItem>
    )
  }

  return (
    <Collapsible
      defaultOpen={normalizedHref.startsWith('/embed')}
      className='group/collapsible'
      render={<SidebarMenuItem />}
    >
      <CollapsibleTrigger
        className='group/collapsible-trigger'
        render={<SidebarMenuButton />}
      >
        {item.icon && <item.icon className='shrink-0' />}
        <span className='min-w-0 flex-1 truncate'>{item.title}</span>
        <ChevronRight className='ms-auto size-4 shrink-0 transition-transform duration-200 group-data-[panel-open]/collapsible-trigger:rotate-90' />
      </CollapsibleTrigger>
      <CollapsibleContent className='CollapsibleContent'>
        <SidebarMenuSub>
          {customTabs.map((tab) => (
            <SidebarMenuSubItem key={tab.id}>
              <SidebarMenuSubButton
                isActive={normalizedHref === `/embed/${tab.id}`}
                render={
                  <Link
                    to='/embed/$tabId'
                    params={{ tabId: tab.id }}
                    onClick={() => setOpenMobile(false)}
                  />
                }
              >
                <span className='min-w-0 flex-1 truncate whitespace-nowrap'>
                  {tab.name}
                </span>
              </SidebarMenuSubButton>
            </SidebarMenuSubItem>
          ))}
        </SidebarMenuSub>
      </CollapsibleContent>
    </Collapsible>
  )
}
