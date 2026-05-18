import { ChatPage } from "@/features/chat/pages/ChatPage"

interface PageProps {
  params: Promise<{ conversationId: string }>
}

export default async function Page({ params }: PageProps) {
  const { conversationId } = await params
  return <ChatPage conversationId={conversationId} />
}
