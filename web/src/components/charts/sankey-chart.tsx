/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { VChart as BaseVChart } from '@visactor/vchart/esm/core'
import { registerSankeyChart } from '@visactor/vchart/esm/chart/sankey'
import { registerDiscreteLegend } from '@visactor/vchart/esm/component/legend/discrete/legend'
import { registerTooltip } from '@visactor/vchart/esm/component/tooltip/tooltip'
import { registerLabel } from '@visactor/vchart/esm/component/label/label'
import {
  registerCanvasTooltipHandler,
  registerDomTooltipHandler,
} from '@visactor/vchart/esm/plugin/components/tooltip-handler'
import {
  registerAnimate,
  registerHtmlAttributePlugin,
  registerReactAttributePlugin,
} from '@visactor/vchart/esm/plugin/other'
import type {
  EventParamsDefinition,
  IInitOption,
  ISpec,
  IVChart,
} from '@visactor/vchart'
import { useEffect, useRef } from 'react'

const registerChartModules = BaseVChart.useRegisters.bind(BaseVChart)

registerChartModules([
  registerSankeyChart,
  registerDiscreteLegend,
  registerTooltip,
  registerLabel,
  registerCanvasTooltipHandler,
  registerDomTooltipHandler,
  registerAnimate,
  registerReactAttributePlugin,
  registerHtmlAttributePlugin,
])

interface SankeyChartProps {
  spec: object
  option: Omit<IInitOption, 'autoFit' | 'dom'>
  onReady?: (instance: IVChart) => void
  onPointerDown?: (event: EventParamsDefinition['pointerdown']) => void
}

export function SankeyChart(props: SankeyChartProps) {
  const containerRef = useRef<HTMLDivElement>(null)
  const { onPointerDown, onReady, option, spec } = props

  useEffect(() => {
    const container = containerRef.current
    if (!container) return

    const chart = new BaseVChart(spec as ISpec, {
      ...option,
      autoFit: true,
      dom: container,
    })
    if (onPointerDown) {
      chart.on('pointerdown', (event) =>
        onPointerDown(event as EventParamsDefinition['pointerdown'])
      )
    }
    onReady?.(chart)
    chart.renderSync({ reuse: false })

    return () => chart.release()
  }, [onPointerDown, onReady, option, spec])

  return <div ref={containerRef} className='relative h-full w-full' />
}
