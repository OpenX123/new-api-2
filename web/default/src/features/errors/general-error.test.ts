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
import assert from 'node:assert/strict'
import { describe, it } from 'node:test'

import { isResourceLoadError } from './general-error-utils'

describe('isResourceLoadError', () => {
  it('recognizes browser module loading failures', () => {
    for (const error of [
      { name: 'ChunkLoadError', message: 'Loading chunk 3444 failed.' },
      { message: 'Failed to fetch dynamically imported module: /route.js' },
      { message: 'Importing a module script failed.' },
    ]) {
      assert.equal(isResourceLoadError(error), true)
    }
  })

  it('does not classify an HTTP error as a resource failure', () => {
    assert.equal(
      isResourceLoadError({
        response: { status: 500 },
        message: 'Request failed',
      }),
      false
    )
  })
})
