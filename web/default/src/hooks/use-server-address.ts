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
import { useMemo } from 'react'

import { useStatus } from '@/hooks/use-status'

/**
 * Public base address of this deployment, used to fill the `{address}`
 * placeholder in admin-configured links (chat presets, custom tabs).
 *
 * Falls back to the browser origin when the operator has not set
 * `ServerAddress` in system settings.
 */
export function useServerAddress(): string {
  const { status } = useStatus()

  return useMemo(() => {
    const fromStatus =
      (status?.server_address as string | undefined) ??
      (status?.serverAddress as string | undefined) ??
      status?.data?.server_address ??
      (status?.data as Record<string, unknown> | undefined)?.serverAddress

    if (fromStatus && typeof fromStatus === 'string') {
      return fromStatus
    }

    if (typeof window !== 'undefined') {
      return window.location.origin
    }

    return ''
  }, [status])
}
