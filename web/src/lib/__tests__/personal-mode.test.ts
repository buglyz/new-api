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
import { describe, test } from 'node:test'

import {
  filterPersonalModeNavGroups,
  getPersonalModeRedirect,
  isPersonalModeEnabled,
  isPersonalModeNavPathDisabled,
  prioritizePersonalModeActions,
  selectPersonalModeTopNavLinks,
} from '../personal-mode'

describe('personal mode decisions', () => {
  test('leaves upstream route and navigation behavior unchanged when disabled', () => {
    const groups = [
      { title: 'Personal', items: [{ title: 'Wallet', url: '/wallet' }] },
    ]
    const links = [{ title: 'Home', href: '/' }]

    assert.equal(getPersonalModeRedirect('/wallet', true, false), null)
    assert.equal(filterPersonalModeNavGroups(groups, false), groups)
    assert.equal(
      selectPersonalModeTopNavLinks(
        links,
        { title: 'Console', href: '/dashboard' },
        false
      ),
      links
    )
  })

  test('uses the direct status contract without accepting wrapper drift', () => {
    assert.equal(isPersonalModeEnabled(undefined), false)
    assert.equal(isPersonalModeEnabled({ self_use_mode_enabled: false }), false)
    assert.equal(isPersonalModeEnabled({ self_use_mode_enabled: true }), true)
    assert.equal(
      isPersonalModeEnabled({ data: { self_use_mode_enabled: true } }),
      false
    )
  })

  test('redirects root and disabled public routes by authentication state', () => {
    assert.equal(getPersonalModeRedirect('/', false, true), '/sign-in')
    assert.equal(getPersonalModeRedirect('/', true, true), '/dashboard')

    for (const path of [
      '/register',
      '/sign-up',
      '/forgot-password',
      '/reset',
      '/user/reset',
      '/oauth',
      '/oauth/github',
      '/pricing/gpt-4o',
      '/rankings',
      '/about',
      '/user-agreement',
      '/privacy-policy',
    ]) {
      assert.equal(getPersonalModeRedirect(path, false, true), '/sign-in', path)
      assert.equal(
        getPersonalModeRedirect(path, true, true),
        '/dashboard',
        path
      )
    }
  })

  test('redirects disabled console and settings routes', () => {
    for (const path of [
      '/users',
      '/wallet/',
      '/subscriptions',
      '/redemption-codes',
      '/dashboard/users',
    ]) {
      assert.equal(
        getPersonalModeRedirect(path, true, true),
        '/dashboard',
        path
      )
    }

    assert.equal(
      getPersonalModeRedirect('/system-settings/billing/payment', true, true),
      '/system-settings/site'
    )
    assert.equal(
      getPersonalModeRedirect('/system-settings/content/faq', true, true),
      '/system-settings/site'
    )
    assert.equal(
      getPersonalModeRedirect('/system-settings/content/dashboard', true, true),
      null
    )
    assert.equal(
      getPersonalModeRedirect('/system-settings/auth/oauth', true, true),
      '/system-settings/auth/passkey'
    )
    assert.equal(
      getPersonalModeRedirect(
        '/system-settings/site/header-navigation',
        true,
        true
      ),
      '/system-settings/site'
    )
    assert.equal(
      getPersonalModeRedirect('/system-settings/models/grok', true, true),
      '/system-settings/models/routing-reliability'
    )
  })

  test('does not redirect allowed operational routes', () => {
    for (const path of [
      '/sign-in',
      '/otp',
      '/dashboard',
      '/profile',
      '/channels',
      '/models',
      '/keys',
      '/usage-logs',
      '/system-info',
      '/system-settings/site',
      '/system-settings/models',
      '/system-settings/security',
      '/system-settings/operations',
      '/system-settings/content/dashboard',
      '/system-settings/auth/passkey',
      '/setup',
    ]) {
      assert.equal(getPersonalModeRedirect(path, true, true), null, path)
    }
  })

  test('keeps only Console in the top navigation', () => {
    const consoleLink = { title: 'Console', href: '/dashboard' }
    const result = selectPersonalModeTopNavLinks(
      [
        { title: 'Home', href: '/' },
        consoleLink,
        { title: 'Pricing', href: '/pricing' },
      ],
      consoleLink,
      true
    )

    assert.deepEqual(result, [consoleLink])
  })

  test('prioritizes failover operations without changing standard mode', () => {
    const actions = [
      { to: '/keys' },
      { to: '/channels' },
      { to: '/usage-logs' },
    ]

    assert.equal(prioritizePersonalModeActions(actions, false), actions)
    assert.deepEqual(prioritizePersonalModeActions(actions, true), [
      { to: '/channels' },
      { to: '/usage-logs' },
      { to: '/keys' },
    ])
  })

  test('filters commercial root links and restricted settings sections', () => {
    const groups = [
      {
        id: 'personal',
        title: 'Personal',
        items: [
          { title: 'Wallet', url: '/wallet' },
          { title: 'Profile', url: '/profile' },
        ],
      },
      {
        id: 'settings',
        title: 'Settings',
        items: [
          {
            title: 'Authentication',
            items: [
              { title: 'Basic', url: '/system-settings/auth/basic-auth' },
              { title: 'Passkey', url: '/system-settings/auth/passkey' },
              { title: 'OAuth', url: '/system-settings/auth/oauth' },
            ],
          },
          {
            title: 'Billing',
            items: [
              { title: 'Payment', url: '/system-settings/billing/payment' },
            ],
          },
          {
            title: 'Console Content',
            items: [
              {
                title: 'Data Dashboard',
                url: '/system-settings/content/dashboard',
              },
              { title: 'FAQ', url: '/system-settings/content/faq' },
            ],
          },
          {
            title: 'Operations',
            items: [
              {
                title: 'Performance',
                url: '/system-settings/operations/performance',
              },
            ],
          },
        ],
      },
    ]

    const result = filterPersonalModeNavGroups(groups, true)

    assert.deepEqual(result, [
      {
        id: 'personal',
        title: 'Personal',
        items: [{ title: 'Profile', url: '/profile' }],
      },
      {
        id: 'settings',
        title: 'Settings',
        items: [
          {
            title: 'Authentication',
            items: [{ title: 'Passkey', url: '/system-settings/auth/passkey' }],
          },
          {
            title: 'Console Content',
            items: [
              {
                title: 'Data Dashboard',
                url: '/system-settings/content/dashboard',
              },
            ],
          },
          {
            title: 'Operations',
            items: [
              {
                title: 'Performance',
                url: '/system-settings/operations/performance',
              },
            ],
          },
        ],
      },
    ])
  })

  test('marks wallet and restricted settings navigation as disabled', () => {
    assert.equal(isPersonalModeNavPathDisabled('/wallet'), true)
    assert.equal(
      isPersonalModeNavPathDisabled('/system-settings/auth/oauth'),
      true
    )
    assert.equal(
      isPersonalModeNavPathDisabled('/system-settings/auth/passkey'),
      false
    )
    assert.equal(
      isPersonalModeNavPathDisabled('/system-settings/content/dashboard'),
      false
    )
    assert.equal(
      isPersonalModeNavPathDisabled('/system-settings/site/header-navigation'),
      true
    )
    assert.equal(
      isPersonalModeNavPathDisabled('/system-settings/models/grok'),
      true
    )
    assert.equal(isPersonalModeNavPathDisabled('/channels'), false)
  })
})
