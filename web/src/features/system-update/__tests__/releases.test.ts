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
import { describe, expect, test } from 'vitest'

import {
  compareSystemVersions,
  formatXXVersionLabel,
  getSystemReleaseUrl,
  selectLatestRelease,
  selectLatestXXRelease,
} from '../releases'

describe('system release ordering', () => {
  test.each([
    ['v0.13.2', 'v1.0.0-alpha.1', -1],
    ['v1.0.0-alpha.2', 'v1.0.0-beta.1', -1],
    ['v1.0.0-beta.1', 'v1.0.0-rc.1', -1],
    ['v1.0.0-rc.9', 'v1.0.0-rc.10', -1],
    ['v1.0.0-rc.19', 'v1.0.0-rc.19-i18nfix.1', -1],
    ['v1.0.0-rc.19-i18nfix.2', 'v1.0.0-rc.20', -1],
    ['v1.0.0-rc.36', 'v1.0.0', -1],
    ['v0.13.1', 'v0.13.1-patch.1', -1],
    ['v0.13.1-patch.9', 'v0.13.1-patch.10', -1],
    ['v0.13.1-patch.10', 'v0.13.2', -1],
    ['v0.8.8.3.2', 'v0.8.8.3.3', -1],
    ['v0.9.3.0', 'v0.9.3', 0],
    ['v1.0.0+build.1', '1.0.0+build.2', 0],
    ['v1.0.0-rc.37', 'v1.0.0-rc.36', 1],
    ['v1.0.0', 'v1.0.0', 0],
    ['v0.0.0', 'v1.0.0', null],
    ['dev', 'v1.0.0', null],
    ['', 'v1.0.0', null],
    ['v1.0.0-custom.1', 'v1.0.0', null],
    ['v1.0.0-rc.36-2-gabcdef', 'v1.0.0', null],
    ['v1.0.0-rc.44-x-x.1', 'v1.0.0-rc.44-x-x.2', -1],
    ['v1.0.0-rc.44-x-x.9', 'v1.0.0-rc.45-x-x.1', -1],
    ['v1.0.0-rc.44-x-x.2+abc1234', 'v1.0.0-rc.44-x-x.2', 0],
    ['v1.0.0-x-x.1', 'v1.0.0-rc.99-x-x.8', 1],
    ['v1.0.0-rc.44-x-x.6a2f7d1ce4e', 'v1.0.0-rc.44-x-x.1', null],
  ])('compares %s against %s as %s', (current, latest, expected) => {
    expect(compareSystemVersions(current, latest)).toBe(expected)
  })

  test('selects the highest published version including pre-releases, regardless of order', () => {
    const releases = [
      { tag_name: 'v1.0.0-rc.19-i18nfix.2', draft: false, prerelease: true },
      { tag_name: 'v0.13.2', draft: false, prerelease: false },
      { tag_name: 'v2.0.0', draft: true, prerelease: false },
      { tag_name: 'v1.0.0-rc.36', draft: false, prerelease: true },
      { tag_name: 'nightly', draft: false, prerelease: true },
    ]
    expect(selectLatestRelease(releases)?.tag_name).toBe('v1.0.0-rc.36')
  })

  test('returns no release for an empty list or a list containing only drafts', () => {
    expect(selectLatestRelease([])).toBeNull()
    expect(
      selectLatestRelease([
        { tag_name: 'v2.0.0', draft: true, prerelease: false },
      ])
    ).toBeNull()
  })

  test('rejects malformed payloads instead of reporting that the system is current', () => {
    expect(() => selectLatestRelease({ message: 'bad response' })).toThrow()
    expect(() => selectLatestRelease([{ tag_name: 42 }])).toThrow()
  })
})

describe('X-X published image releases', () => {
  function imageRelease(version: string) {
    const tag = `x-x-custom-${version}`
    const archive = `new-api-x-x-${tag}-linux-amd64.tar.gz`
    return {
      tag_name: tag,
      draft: false,
      prerelease: true,
      assets: [
        { name: archive, state: 'uploaded', size: 100 },
        { name: `${archive}.sha256`, state: 'uploaded', size: 100 },
      ],
    }
  }

  test('selects only the newest complete custom image, not official tags, drafts or incomplete builds', () => {
    const ready = imageRelease('v1.0.0-rc.44-x-x.2')
    const incomplete = imageRelease('v1.0.0-rc.45-x-x.1')
    incomplete.assets.pop()
    const uploading = imageRelease('v1.0.0-rc.46-x-x.1')
    uploading.assets[0].state = 'new'
    expect(
      selectLatestXXRelease([
        { ...ready, tag_name: 'v99.0.0' },
        { ...imageRelease('v2.0.0-x-x.1'), draft: true },
        incomplete,
        uploading,
        ready,
        imageRelease('v1.0.0-rc.44-x-x.1'),
      ])?.tag_name
    ).toBe(ready.tag_name)
  })

  test('returns no custom update for old date tags or official-only releases', () => {
    expect(
      selectLatestXXRelease([
        { tag_name: 'x-x-custom-20260810', draft: false, prerelease: false },
        { tag_name: 'v2.0.0', draft: false, prerelease: false },
      ])
    ).toBeNull()
  })

  test('compares release tags and running build versions using the same numeric revision', () => {
    expect(
      compareSystemVersions(
        'v1.0.0-rc.44-x-x.1',
        'x-x-custom-v1.0.0-rc.44-x-x.2'
      )
    ).toBe(-1)
    expect(formatXXVersionLabel('v1.0.0-rc.44-x-x.2')).toBe('X-X rc.44 · 2')
    expect(formatXXVersionLabel('v1.0.0-x-x.3')).toBe('X-X v1.0.0 · 3')
    expect(formatXXVersionLabel('dev')).toBe('dev')
  })

  test('links custom releases to the fork while preserving the official release link', () => {
    const release = imageRelease('v1.0.0-rc.44-x-x.2')
    expect(getSystemReleaseUrl(release, 'x-x')).toBe(
      'https://github.com/lqy007700/new-api/releases/tag/x-x-custom-v1.0.0-rc.44-x-x.2'
    )
    expect(getSystemReleaseUrl({ ...release, tag_name: 'v1.0.0-rc.44' })).toBe(
      'https://github.com/QuantumNous/new-api/releases/tag/v1.0.0-rc.44'
    )
  })
})
