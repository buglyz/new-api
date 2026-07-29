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
import type { IInitOption, ISpec } from '@visactor/vchart'
import VChartSimple from '@visactor/vchart/esm/vchart-simple'
import { useEffect, useRef } from 'react'

interface ModelChartProps {
  spec: ISpec
  option: Omit<IInitOption, 'autoFit' | 'dom'>
}

export function ModelChart(props: ModelChartProps) {
  const containerRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const container = containerRef.current
    if (!container) return

    const chart = new VChartSimple(props.spec, {
      ...props.option,
      autoFit: true,
      dom: container,
    })
    chart.renderSync({ reuse: false })

    return () => chart.release()
  }, [props.option, props.spec])

  return <div ref={containerRef} className='relative h-full w-full' />
}
