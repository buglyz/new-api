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
type StatusLike = Record<string, unknown> | null | undefined

type NavigationChild = {
  url?: unknown
}

type NavigationItem = {
  url?: unknown
  items?: NavigationChild[]
}

type NavigationGroup = {
  items: NavigationItem[]
}

const PUBLIC_DISABLED_PATHS = new Set([
  '/register',
  '/sign-up',
  '/forgot-password',
  '/reset',
  '/user/reset',
  '/oauth',
  '/pricing',
  '/rankings',
  '/about',
  '/user-agreement',
  '/privacy-policy',
])

const CONSOLE_DISABLED_PATHS = new Set([
  '/users',
  '/wallet',
  '/subscriptions',
  '/redemption-codes',
])

const HIDDEN_ROOT_NAV_PATHS = new Set([
  '/users',
  '/wallet',
  '/subscriptions',
  '/redemption-codes',
])

const PERSONAL_MODE_ACTION_PRIORITY = new Map([
  ['/channels', 0],
  ['/usage-logs', 1],
  ['/keys', 2],
])

function normalizePath(pathname: string): string {
  if (!pathname || pathname === '/') return '/'
  return pathname.replace(/\/+$/, '') || '/'
}

function isPathOrChild(pathname: string, basePath: string): boolean {
  return pathname === basePath || pathname.startsWith(`${basePath}/`)
}

export function isPersonalModeEnabled(status: StatusLike): boolean {
  return status?.self_use_mode_enabled === true
}

export function getPersonalModeRedirect(
  pathname: string,
  authenticated: boolean,
  personalMode: boolean
): string | null {
  if (!personalMode) return null

  const path = normalizePath(pathname)
  if (path === '/') return authenticated ? '/dashboard' : '/sign-in'

  const publicDisabled =
    PUBLIC_DISABLED_PATHS.has(path) ||
    isPathOrChild(path, '/oauth') ||
    isPathOrChild(path, '/pricing') ||
    isPathOrChild(path, '/rankings') ||
    isPathOrChild(path, '/about')
  if (publicDisabled) return authenticated ? '/dashboard' : '/sign-in'

  const consoleDisabled =
    CONSOLE_DISABLED_PATHS.has(path) || isPathOrChild(path, '/dashboard/users')
  if (consoleDisabled) return authenticated ? '/dashboard' : '/sign-in'
  if (
    isPathOrChild(path, '/system-settings/billing') ||
    (isPathOrChild(path, '/system-settings/content') &&
      path !== '/system-settings/content/dashboard')
  ) {
    return '/system-settings/site'
  }
  if (
    isPathOrChild(path, '/system-settings/auth') &&
    path !== '/system-settings/auth/passkey'
  ) {
    return '/system-settings/auth/passkey'
  }
  if (
    path === '/system-settings/site/header-navigation' ||
    path === '/system-settings/site/sidebar-modules'
  ) {
    return '/system-settings/site'
  }
  if (path === '/system-settings/models/grok') {
    return '/system-settings/models/routing-reliability'
  }

  return null
}

export function selectPersonalModeTopNavLinks<T>(
  links: T[],
  consoleLink: T,
  personalMode: boolean
): T[] {
  return personalMode ? [consoleLink] : links
}

export function prioritizePersonalModeActions<T extends { to: string }>(
  actions: T[],
  personalMode: boolean
): T[] {
  if (!personalMode) return actions
  return [...actions].sort((left, right) => {
    const leftPriority = PERSONAL_MODE_ACTION_PRIORITY.get(left.to) ?? 100
    const rightPriority = PERSONAL_MODE_ACTION_PRIORITY.get(right.to) ?? 100
    return leftPriority - rightPriority
  })
}

export function isPersonalModeNavPathDisabled(url: string): boolean {
  if (HIDDEN_ROOT_NAV_PATHS.has(normalizePath(url))) return true
  if (isPathOrChild(url, '/system-settings/billing')) return true
  if (
    isPathOrChild(url, '/system-settings/content') &&
    normalizePath(url) !== '/system-settings/content/dashboard'
  ) {
    return true
  }
  if (
    normalizePath(url) === '/system-settings/site/header-navigation' ||
    normalizePath(url) === '/system-settings/site/sidebar-modules' ||
    normalizePath(url) === '/system-settings/models/grok'
  ) {
    return true
  }
  return (
    isPathOrChild(url, '/system-settings/auth') &&
    normalizePath(url) !== '/system-settings/auth/passkey'
  )
}

function filterNavItem<T extends NavigationItem>(item: T): T | null {
  if (item.items) {
    const items = item.items.filter((child) =>
      child.url ? !isPersonalModeNavPathDisabled(String(child.url)) : true
    )
    return items.length > 0 ? ({ ...item, items } as T) : null
  }

  if (item.url) {
    return isPersonalModeNavPathDisabled(String(item.url)) ? null : item
  }
  return item
}

export function filterPersonalModeNavGroups<T extends NavigationGroup>(
  groups: T[],
  personalMode: boolean
): T[] {
  if (!personalMode) return groups

  return groups.flatMap((group) => {
    const items = group.items.flatMap((item) => {
      const filtered = filterNavItem(item)
      return filtered ? [filtered] : []
    })
    return items.length > 0 ? [{ ...group, items } as T] : []
  })
}
