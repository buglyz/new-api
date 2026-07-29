import { createFileRoute, redirect } from '@tanstack/react-router'
import z from 'zod'

import { Pricing } from '@/features/pricing'
import { useAuthStore } from '@/stores/auth-store'

const modelSquareSearchSchema = z.object({
  search: z.string().optional(),
  sort: z.string().optional(),
  vendor: z.string().optional(),
  group: z.string().optional(),
  quotaType: z.string().optional(),
  endpointType: z.string().optional(),
  tag: z.string().optional(),
  tokenUnit: z.enum(['M', 'K']).optional(),
  view: z.enum(['card', 'table']).optional().catch(undefined),
  rechargePrice: z.boolean().optional(),
})

export const Route = createFileRoute('/pricing/')({
  validateSearch: modelSquareSearchSchema,
  beforeLoad: ({ location }) => {
    if (!useAuthStore.getState().auth.user) {
      throw redirect({
        to: '/sign-in',
        search: { redirect: location.href },
      })
    }
  },
  component: Pricing,
})
