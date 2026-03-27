"use client";

import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";

export interface PrepEntryButtonProps {
  leadID: string;
  disabled?: boolean;
}

export function PrepEntryButton({ leadID, disabled = false }: PrepEntryButtonProps) {
  const router = useRouter();
  const normalizedLeadID = (leadID || "").trim();

  return (
    <Button
      type="button"
      variant="secondary"
      size="sm"
      className="h-7 px-2 text-xs"
      disabled={disabled || !normalizedLeadID}
      onClick={() => {
        if (!normalizedLeadID) {
          return;
        }
        router.push(`/prep?lead_id=${encodeURIComponent(normalizedLeadID)}`);
      }}
    >
      备面
    </Button>
  );
}
