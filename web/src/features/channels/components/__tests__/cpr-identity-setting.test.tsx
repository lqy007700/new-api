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
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useForm } from 'react-hook-form'
import { describe, expect, test } from 'vitest'

import { Button } from '@/components/ui/button'
import { Form } from '@/components/ui/form'

import {
  buildSettingJSON,
  CHANNEL_FORM_DEFAULT_VALUES,
  transformChannelToFormDefaults,
  transformFormDataToCreatePayload,
  transformFormDataToUpdatePayload,
  type ChannelFormValues,
} from '../../lib/channel-form'
import { channelSchema } from '../../types'
import { CPRIdentitySetting } from '../cpr-identity-setting'

function SettingsForm(props: {
  channelType: number
  disabled?: boolean
  onSave: (value: string) => void
}) {
  const form = useForm<ChannelFormValues>({
    defaultValues: { ...CHANNEL_FORM_DEFAULT_VALUES, type: props.channelType },
  })
  return (
    <Form {...form}>
      <form
        onSubmit={form.handleSubmit((values) =>
          props.onSave(buildSettingJSON(values))
        )}
      >
        <CPRIdentitySetting
          channelType={props.channelType}
          disabled={props.disabled}
        />
        <Button type='submit'>Save</Button>
      </form>
    </Form>
  )
}

describe('CPR identity forwarding', () => {
  test.each([undefined, false, true])(
    'preserves %s through create, update and reload',
    (enabled) => {
      const channel = channelSchema.parse({
        id: 1,
        name: 'CPR',
        key: '',
        type: 1,
        status: 1,
        created_time: 0,
        test_time: 0,
        response_time: 0,
        balance_updated_time: 0,
        setting: JSON.stringify({
          cpr_identity_forwarding_enabled: enabled,
          system_prompt: 'preserved',
        }),
      })
      const defaults = transformChannelToFormDefaults(channel)
      expect(defaults.cpr_identity_forwarding_enabled).toBe(enabled === true)
      for (const payload of [
        transformFormDataToCreatePayload(defaults).channel,
        transformFormDataToUpdatePayload(defaults, channel.id),
      ]) {
        const settings = JSON.parse(payload.setting ?? '{}')
        expect(settings.cpr_identity_forwarding_enabled).toBe(
          enabled === true ? true : undefined
        )
        expect(settings.system_prompt).toBe('preserved')
        expect(
          transformChannelToFormDefaults({
            ...channel,
            setting: payload.setting,
          }).cpr_identity_forwarding_enabled
        ).toBe(enabled === true)
      }
    }
  )

  test('defaults off and saves explicit opt-in', async () => {
    let saved = ''
    render(
      <SettingsForm
        channelType={1}
        onSave={(value) => {
          saved = value
        }}
      />
    )
    const toggle = screen.getByRole('switch', {
      name: 'Forward trusted identity to CPR',
    })
    expect(toggle).not.toBeChecked()
    const user = userEvent.setup()
    await user.click(toggle)
    await user.click(screen.getByRole('button', { name: 'Save' }))
    expect(JSON.parse(saved).cpr_identity_forwarding_enabled).toBe(true)
  })

  test('respects sensitive permission lock', async () => {
    render(<SettingsForm channelType={1} disabled onSave={() => {}} />)
    const toggle = screen.getByRole('switch')
    expect(toggle).toHaveAttribute('aria-disabled', 'true')
    await userEvent.setup().click(toggle)
    expect(toggle).not.toBeChecked()
  })

  test.each([2, 14, 57, 58, 59, 60, 61])(
    'hides and omits identity forwarding for channel type %s',
    (channelType) => {
      render(<SettingsForm channelType={channelType} onSave={() => {}} />)
      expect(screen.queryByRole('switch')).not.toBeInTheDocument()
      expect(
        JSON.parse(
          buildSettingJSON({
            ...CHANNEL_FORM_DEFAULT_VALUES,
            type: channelType,
            cpr_identity_forwarding_enabled: true,
          })
        ).cpr_identity_forwarding_enabled
      ).toBeUndefined()
    }
  )
})
