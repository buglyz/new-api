import { createFileRoute, redirect } from '@tanstack/react-router'
import z from 'zod'

import { ModelDetails } from '@/features/pricing/components/model-details'
import { useAuthStore } from '@/stores/auth-store'

const modelDetailsSearchSchema = z.object({
  search: z.string().optional(),
  sort: z.string().optional(),
  vendor: z.string().optional(),
  group: z.string().optional(),
  quotaType: z.string().optional(),
  endpointType: z.string().optional(),
  tag: z.string().optional(),
  tokenUnit: z.enum(['M', 'K']).optional(),
  view: z.enum(['card', 'table']).optional().catch(undefined),
})

export const Route = createFileRoute('/pricing/$modelId/')({
  validateSearch: modelDetailsSearchSchema,
  beforeLoad: ({ location }) => {
    if (!useAuthStore.getState().auth.user) {
      throw redirect({
        to: '/sign-in',
        search: { redirect: location.href },
      })
    }
  },
  component: ModelDetails,
})
