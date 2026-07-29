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
import { Link } from '@tanstack/react-router'
import { LogOut, User } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { SignOutDialog } from '@/components/sign-out-dialog'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Button } from '@/components/ui/button'
import useDialogState from '@/hooks/use-dialog'
import { useUserDisplay } from '@/hooks/use-user-display'
import type { AuthUser } from '@/stores/auth-store'

interface MobileUserProfileProps {
  user: AuthUser
  onNavigate?: () => void
}

export function MobileUserProfile(props: MobileUserProfileProps) {
  const { t } = useTranslation()
  const [signOutOpen, setSignOutOpen] = useDialogState()
  const { displayName, initials, roleLabel } = useUserDisplay(props.user)

  return (
    <>
      <div className='flex flex-col text-sm'>
        <div className='border-border flex items-center gap-2.5 border-b p-2.5'>
          <Avatar className='size-9'>
            <AvatarImage src='/avatars/01.png' alt={`@${displayName}`} />
            <AvatarFallback className='text-xs'>{initials}</AvatarFallback>
          </Avatar>
          <div className='flex flex-1 flex-col gap-0.5 overflow-hidden'>
            <p className='text-foreground truncate font-medium'>
              {displayName}
            </p>
            <div className='flex items-center gap-1.5'>
              <span className='text-muted-foreground text-xs'>{roleLabel}</span>
              {props.user.group && (
                <>
                  <span className='text-muted-foreground text-xs'>·</span>
                  <span className='text-muted-foreground text-xs'>
                    {String(props.user.group)}
                  </span>
                </>
              )}
            </div>
          </div>
        </div>

        <Link
          to='/profile'
          onClick={props.onNavigate}
          className='text-primary/60 hover:text-primary/80 border-border flex items-center gap-2.5 border-b p-2.5 transition-colors'
        >
          <User className='size-4' />
          {t('Profile')}
        </Link>

        <Button
          variant='ghost'
          onClick={() => setSignOutOpen(true)}
          className='text-destructive hover:text-destructive/80 h-auto w-full justify-start gap-2.5 p-2.5 hover:bg-transparent'
        >
          <LogOut className='size-4' />
          {t('Sign out')}
        </Button>
      </div>

      <SignOutDialog open={!!signOutOpen} onOpenChange={setSignOutOpen} />
    </>
  )
}
