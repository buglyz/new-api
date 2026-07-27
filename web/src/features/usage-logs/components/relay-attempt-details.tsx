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
import { useTranslation } from 'react-i18next'

import { getAttemptOutcomeLabelKey, type FailoverTrace } from '../lib/failover'

export function RelayAttemptDetails(props: { trace: FailoverTrace }) {
  const { t } = useTranslation()
  return (
    <span className='space-y-1'>
      {props.trace.attempts.map((attempt) => (
        <span
          key={`${attempt.channelId}-${attempt.index}`}
          className='block break-all'
        >
          #{attempt.channelId}
          {attempt.outcome
            ? ` · ${t(getAttemptOutcomeLabelKey(attempt.outcome))}`
            : ''}
          {attempt.statusCode != null ? ` · HTTP ${attempt.statusCode}` : ''}
          {attempt.durationMs != null ? ` · ${attempt.durationMs} ms` : ''}
        </span>
      ))}
    </span>
  )
}
