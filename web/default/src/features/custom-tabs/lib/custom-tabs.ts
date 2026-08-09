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

/**
 * Custom tabs are admin-configured pages embedded into the console sidebar as
 * iframes. Unlike chat presets they are always plain web pages — custom URL
 * protocols cannot be framed, so non-HTTP entries are dropped while parsing.
 *
 * URL templates support the same `{key}` / `{address}` placeholders as chat
 * presets and are resolved through `resolveChatUrl`.
 */
export type CustomTab = {
  id: string
  name: string
  url: string
}

export type RawCustomTabsConfig =
  | string
  | Array<Record<string, unknown>>
  | null
  | undefined

const HTTP_REGEX = /^https?:\/\//i

export function parseCustomTabsConfig(raw: RawCustomTabsConfig): CustomTab[] {
  let parsed: unknown = raw

  if (typeof raw === 'string') {
    try {
      parsed = JSON.parse(raw)
    } catch {
      return []
    }
  }

  if (!Array.isArray(parsed)) {
    return []
  }

  return parsed
    .map((entry, index) => {
      if (!entry || typeof entry !== 'object' || Array.isArray(entry)) {
        return null
      }

      const { name, url } = entry as Record<string, unknown>
      if (typeof name !== 'string' || typeof url !== 'string') {
        return null
      }

      const trimmedName = name.trim()
      const trimmedUrl = url.trim()
      if (!trimmedName || !HTTP_REGEX.test(trimmedUrl)) {
        return null
      }

      return {
        id: String(index),
        name: trimmedName,
        url: trimmedUrl,
      } satisfies CustomTab
    })
    .filter((item): item is CustomTab => item !== null)
}
