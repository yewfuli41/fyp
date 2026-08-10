// Small inline-SVG icon set used across the customer booking flow — kept
// dependency-free (no icon package) since these are the only few glyphs
// the redesigned pages need.
type IconProps = { size?: number; className?: string };

const base = (size: number) => ({
    width: size,
    height: size,
    viewBox: "0 0 24 24",
    fill: "none",
    stroke: "currentColor",
    strokeWidth: 2,
    strokeLinecap: "round" as const,
    strokeLinejoin: "round" as const,
});

export const IconSearch = ({ size = 18, className }: IconProps) => (
    <svg {...base(size)} className={className}>
        <circle cx="11" cy="11" r="7" />
        <line x1="21" y1="21" x2="16.65" y2="16.65" />
    </svg>
);

export const IconPin = ({ size = 16, className }: IconProps) => (
    <svg {...base(size)} className={className}>
        <path d="M20 10c0 6-8 12-8 12s-8-6-8-12a8 8 0 0 1 16 0Z" />
        <circle cx="12" cy="10" r="3" />
    </svg>
);

export const IconPhone = ({ size = 16, className }: IconProps) => (
    <svg {...base(size)} className={className}>
        <path d="M22 16.92v3a2 2 0 0 1-2.18 2 19.79 19.79 0 0 1-8.63-3.07 19.5 19.5 0 0 1-6-6 19.79 19.79 0 0 1-3.07-8.67A2 2 0 0 1 4.11 2h3a2 2 0 0 1 2 1.72c.127.96.362 1.903.7 2.81a2 2 0 0 1-.45 2.11L8.09 9.91a16 16 0 0 0 6 6l1.27-1.27a2 2 0 0 1 2.11-.45c.907.338 1.85.573 2.81.7A2 2 0 0 1 22 16.92Z" />
    </svg>
);

export const IconMail = ({ size = 16, className }: IconProps) => (
    <svg {...base(size)} className={className}>
        <rect x="2" y="4" width="20" height="16" rx="2.5" />
        <path d="m2 6.5 9.4 6.4a1 1 0 0 0 1.2 0L22 6.5" />
    </svg>
);

export const IconCalendar = ({ size = 16, className }: IconProps) => (
    <svg {...base(size)} className={className}>
        <rect x="3" y="4.5" width="18" height="16" rx="2.5" />
        <line x1="3" y1="9.5" x2="21" y2="9.5" />
        <line x1="8" y1="2.5" x2="8" y2="6.5" />
        <line x1="16" y1="2.5" x2="16" y2="6.5" />
    </svg>
);

export const IconChevronLeft = ({ size = 16, className }: IconProps) => (
    <svg {...base(size)} className={className}>
        <polyline points="15 18 9 12 15 6" />
    </svg>
);

export const IconChevronRight = ({ size = 16, className }: IconProps) => (
    <svg {...base(size)} className={className}>
        <polyline points="9 18 15 12 9 6" />
    </svg>
);

export const IconCheck = ({ size = 14, className }: IconProps) => (
    <svg {...base(size)} className={className}>
        <polyline points="20 6 9 17 4 12" />
    </svg>
);

export const IconUser = ({ size = 16, className }: IconProps) => (
    <svg {...base(size)} className={className}>
        <circle cx="12" cy="8" r="4" />
        <path d="M4 20c0-4 3.6-6.5 8-6.5s8 2.5 8 6.5" />
    </svg>
);

export const IconClock = ({ size = 16, className }: IconProps) => (
    <svg {...base(size)} className={className}>
        <circle cx="12" cy="12" r="9" />
        <polyline points="12 7 12 12 15.5 14" />
    </svg>
);

export const IconPencil = ({ size = 16, className }: IconProps) => (
    <svg {...base(size)} className={className}>
        <path d="M12 20h9" />
        <path d="M16.5 3.5a2.12 2.12 0 0 1 3 3L7 19l-4 1 1-4Z" />
    </svg>
);

export const IconEye = ({ size = 16, className }: IconProps) => (
    <svg {...base(size)} className={className}>
        <path d="M2 12s3.5-7 10-7 10 7 10 7-3.5 7-10 7-10-7-10-7Z" />
        <circle cx="12" cy="12" r="3" />
    </svg>
);

// Clock face with a counter-clockwise arrow — the conventional "history"
// glyph (time already passed, wound back).
export const IconHistory = ({ size = 16, className }: IconProps) => (
    <svg {...base(size)} className={className}>
        <path d="M3 3v5h5" />
        <path d="M3.05 13A9 9 0 1 0 6 5.3L3 8" />
        <polyline points="12 7 12 12 16 14" />
    </svg>
);

export const IconBell = ({ size = 16, className }: IconProps) => (
    <svg {...base(size)} className={className}>
        <path d="M6 8a6 6 0 0 1 12 0c0 4.5 1.5 6 2 6.5H4c.5-.5 2-2 2-6.5Z" />
        <path d="M10 19a2 2 0 0 0 4 0" />
    </svg>
);

export const IconChart = ({ size = 16, className }: IconProps) => (
    <svg {...base(size)} className={className}>
        <line x1="4" y1="20" x2="20" y2="20" />
        <rect x="6" y="12" width="3.5" height="8" />
        <rect x="13" y="7" width="3.5" height="13" />
    </svg>
);

export const IconUsers = ({ size = 16, className }: IconProps) => (
    <svg {...base(size)} className={className}>
        <circle cx="9" cy="8" r="3.2" />
        <path d="M3 20c0-3.5 2.7-5.5 6-5.5s6 2 6 5.5" />
        <path d="M15.5 5.2a3.2 3.2 0 0 1 0 6.1" />
        <path d="M17 14.6c2.4.5 4 2.2 4 5.4" />
    </svg>
);
