import assert from 'node:assert/strict'
import test from 'node:test'

import { resolveCCSwitchDefaults } from './cc-switch'

test('resolves configured CC Switch defaults and keeps optional mappings empty', () => {
  assert.deepEqual(
    resolveCCSwitchDefaults('claude', {
      claude_name: 'Team Claude',
      claude_model: 'claude-main',
    }),
    {
      name: 'Team Claude',
      models: {
        model: 'claude-main',
        haikuModel: '',
        sonnetModel: '',
        opusModel: '',
      },
    }
  )
})
