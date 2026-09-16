// Shared editable-field states. Component and caller classes own layout.
export const fieldControl =
  "min-w-0 rounded-md border border-input bg-well text-[13px] text-foreground shadow-none outline-none transition-[color,border-color,box-shadow,background-color] placeholder:text-muted-foreground enabled:hover:not-focus-visible:not-aria-invalid:border-muted-foreground focus-visible:border-ring focus-visible:ring-2 focus-visible:ring-ring/50 aria-invalid:border-destructive aria-invalid:ring-2 aria-invalid:ring-destructive/20 disabled:cursor-not-allowed disabled:opacity-50 motion-reduce:transition-none";

export const textControl = `${fieldControl} h-8 w-full px-2.5 py-1`;
export const pickerControl = `${textControl} pr-9`;
