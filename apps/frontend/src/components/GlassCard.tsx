import { cn } from "@/lib/utils";
import { HTMLAttributes, forwardRef } from "react";

interface GlassCardProps extends HTMLAttributes<HTMLDivElement> {
  variant?: "default" | "strong";
  hover?: boolean;
}

export const GlassCard = forwardRef<HTMLDivElement, GlassCardProps>(
  ({ className, variant = "default", hover = false, ...props }, ref) => (
    <div
      ref={ref}
      className={cn(
        "rounded-3xl",
        variant === "strong" ? "glass-strong" : "glass",
        hover && "glass-hover cursor-pointer",
        className
      )}
      {...props}
    />
  )
);
GlassCard.displayName = "GlassCard";