import { Suspense } from "react";
import { PrepWorkspace } from "@/components/prep/prep-workspace";

export default function PrepPage() {
  return (
    <Suspense fallback={null}>
      <PrepWorkspace />
    </Suspense>
  );
}
