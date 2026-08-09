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
import { SettingsPage } from '../components/settings-page'
import type { CustomSettings } from '../types'
import {
  CUSTOM_DEFAULT_SECTION,
  getCustomSectionContent,
  getCustomSectionMeta,
} from './section-registry.tsx'

const defaultCustomSettings: CustomSettings = {
  'cc_switch_setting.claude_name': 'My Claude',
  'cc_switch_setting.claude_model': '',
  'cc_switch_setting.claude_haiku_model': '',
  'cc_switch_setting.claude_sonnet_model': '',
  'cc_switch_setting.claude_opus_model': '',
  'cc_switch_setting.codex_name': 'My Codex',
  'cc_switch_setting.codex_model': '',
  'cc_switch_setting.gemini_name': 'My Gemini',
  'cc_switch_setting.gemini_model': '',
}

export function CustomSettingsPage() {
  return (
    <SettingsPage
      routePath='/_authenticated/system-settings/custom/$section'
      defaultSettings={defaultCustomSettings}
      defaultSection={CUSTOM_DEFAULT_SECTION}
      getSectionContent={getCustomSectionContent}
      getSectionMeta={getCustomSectionMeta}
      loadingMessage='Loading custom configuration...'
    />
  )
}
