import { Chat } from "@/components/chat";

export default function AgentPage() {
  return (
    <div className="flex min-h-0 h-[calc(100dvh-var(--app-nav-height,176px))] flex-col overflow-hidden">
      <main className="mx-auto flex h-full w-full max-w-6xl flex-1 min-h-0 overflow-hidden px-4 pb-4 pt-6 sm:px-6">
        <div className="page-enter flex h-full min-h-0 w-full flex-1 overflow-hidden rounded-[32px] border border-[var(--panel-border)] bg-card/74 shadow-[var(--panel-shadow)] backdrop-blur-xl">
          <Chat />
        </div>
      </main>
    </div>
  );
}
