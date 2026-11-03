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
import { useFormContext } from 'react-hook-form'
import { useTranslation } from 'react-i18next'

import { FormField } from '@/components/ui/form'
import { SettingsSwitchField } from '@/features/system-settings/components/settings-form-layout'

import type { ChannelFormValues } from '../lib/channel-form'

export function CPRIdentitySetting(props: {
  channelType: number
  disabled?: boolean
}) {
  const { t } = useTranslation()
  const form = useFormContext<ChannelFormValues>()
  if (props.channelType !== 1) return null

  return (
    <FormField
      control={form.control}
      name='cpr_identity_forwarding_enabled'
      render={({ field }) => (
        <SettingsSwitchField
          controlId='cpr-identity-forwarding'
          className='px-4 py-3'
          checked={field.value === true}
          onCheckedChange={field.onChange}
          disabled={props.disabled}
          label={t('Forward trusted identity to CPR')}
          description={t(
            'Enable only for CPR upstreams. Sends authenticated user and token IDs, never the user API key. Authorization uses the channel key. Session metadata stays separate. Playground is unavailable when enabled.'
          )}
        />
      )}
    />
  )
}
