/**
 * Example: How to integrate AI Chat Assistant
 *
 * Add this to your layout or any page where you want the floating assistant
 */
'use client'

import { useState } from 'react'
import { FloatingAssistantButton } from '@/components/aiops/FloatingAssistantButton'
import { AIChatDrawer } from '@/components/aiops/AIChatDrawer'

export function AIChatAssistantProvider({ children, currentJobName }: { children: React.ReactNode; currentJobName?: string }) {
  const [isChatOpen, setIsChatOpen] = useState(false)

  return (
    <>
      {children}

      {/* Floating Button - Always visible */}
      <FloatingAssistantButton
        onClick={() => setIsChatOpen(true)}
      />

      {/* Chat Drawer - Opens on click */}
      <AIChatDrawer
        isOpen={isChatOpen}
        onClose={() => setIsChatOpen(false)}
        currentJobName={currentJobName}
      />
    </>
  )
}

/**
 * Usage in your root layout or specific pages:
 *
 * import { AIChatAssistantProvider } from '@/components/aiops/AIChatAssistantProvider'
 *
 * export default function YourPage() {
 *   return (
 *     <AIChatAssistantProvider currentJobName="optional-job-name">
 *       <YourPageContent />
 *     </AIChatAssistantProvider>
 *   )
 * }
 */
