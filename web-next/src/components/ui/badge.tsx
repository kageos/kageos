import * as React from "react";

import { cn } from "@/lib/utils";

export function Badge({
  className,
  ...props
}: React.HTMLAttributes<HTMLSpanElement>) {
  return (
    <span
      className={cn(
        "inline-flex h-6 items-center rounded-md border border-slate-200 bg-white px-2 text-xs font-medium text-slate-600",
        className,
      )}
      {...props}
    />
  );
}
