import { Chat } from "@/components/chat";

export default function AgentPage() {
  return (
    <div className="flex h-full min-h-0 flex-col">
      <main className="mx-auto flex w-full max-w-6xl flex-1 min-h-0 px-4 pb-10 pt-6 sm:px-6">
        <div className="page-enter flex min-h-0 w-full flex-1 overflow-hidden rounded-[32px] border border-[var(--panel-border)] bg-card/74 shadow-[var(--panel-shadow)] backdrop-blur-xl">
          <Chat />
        </div>
      </main>
    </div>
  );
}
