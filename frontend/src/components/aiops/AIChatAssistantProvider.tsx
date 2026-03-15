'use client'

import { useState } from 'react'

import { AIChatDrawer } from '@/components/aiops/AIChatDrawer'
import { FloatingAssistantButton } from '@/components/aiops/FloatingAssistantButton'

export function AIChatAssistantProvider({
  children,
  currentJobName,
}: {
  children: React.ReactNode
  currentJobName?: string
}) {
  const [isChatOpen, setIsChatOpen] = useState(false)

  return (
    <>
      {children}

      {/* Floating Button - Always visible */}
      <FloatingAssistantButton onClick={() => setIsChatOpen(true)} />

      {/* Chat Drawer - Opens on click */}
      <AIChatDrawer
        isOpen={isChatOpen}
        onClose={() => setIsChatOpen(false)}
        currentJobName={currentJobName}
      />
    </>
  )
}
