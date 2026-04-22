import { useMemo } from "react";

/**
 * Frutiger Aero ambience: glossy bubbles drifting upward over a sky/sunset gradient.
 * Pure decoration — pointer-events: none, fixed behind all content.
 */
export const AeroBackground = () => {
  const bubbles = useMemo(
    () =>
      Array.from({ length: 14 }).map((_, i) => {
        const size = 40 + Math.random() * 140;
        return {
          key: i,
          size,
          left: Math.random() * 100,
          delay: -Math.random() * 14,
          duration: 12 + Math.random() * 10,
          orange: Math.random() > 0.7,
        };
      }),
    [],
  );

  return (
    <div aria-hidden className="pointer-events-none fixed inset-0 -z-10 overflow-hidden">
      {/* Sky gradient base */}
      <div
        className="absolute inset-0"
        style={{ background: "var(--gradient-sky)" }}
      />
      {/* Soft mesh on top of sky */}
      <div
        className="absolute inset-0 opacity-90"
        style={{ backgroundImage: "var(--gradient-mesh)" }}
      />
      {/* Glossy bubbles */}
      {bubbles.map((b) => (
        <span
          key={b.key}
          className={`absolute bottom-[-200px] ${b.orange ? "aero-bubble-orange" : "aero-bubble"} animate-rise`}
          style={{
            width: b.size,
            height: b.size,
            left: `${b.left}%`,
            animationDelay: `${b.delay}s`,
            animationDuration: `${b.duration}s`,
          }}
        />
      ))}
      {/* Subtle white veil at top for crispness */}
      <div className="absolute inset-x-0 top-0 h-32 bg-gradient-to-b from-white/40 to-transparent" />
    </div>
  );
};