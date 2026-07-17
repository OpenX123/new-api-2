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
import { describe, expect, it } from 'bun:test'

import { shouldOpenDailyNotice } from './daily-notice'

describe('shouldOpenDailyNotice', () => {
  const today = 'Fri Jul 17 2026'

  it('opens a non-empty notice that was not closed today', () => {
    expect(shouldOpenDailyNotice('Important update', false, null, today)).toBe(
      true
    )
    expect(
      shouldOpenDailyNotice('Important update', false, 'Thu Jul 16 2026', today)
    ).toBe(true)
  })

  it('stays closed after the user closes it on the same local date', () => {
    expect(shouldOpenDailyNotice('Important update', false, today, today)).toBe(
      false
    )
  })

  it('does not open while loading or for empty content', () => {
    expect(shouldOpenDailyNotice('Important update', true, null, today)).toBe(
      false
    )
    expect(shouldOpenDailyNotice('   ', false, null, today)).toBe(false)
  })
})
