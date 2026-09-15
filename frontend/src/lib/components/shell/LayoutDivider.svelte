<script lang="ts">
  let {
    orientation,
    label,
    value,
    min,
    max,
    container,
    onResize,
  }: {
    orientation: "horizontal" | "vertical";
    label: string;
    value: number;
    min: number;
    max: number;
    container: HTMLElement | undefined;
    onResize: (value: number) => void;
  } = $props();
  let dragging = $state(false);
  let origin = 0;
  let initial = 0;
  function update(value: number) {
    onResize(Math.min(max, Math.max(min, value)));
  }
  function start(event: PointerEvent) {
    if (event.button !== 0) return;
    event.preventDefault();
    const target = event.currentTarget as HTMLElement;
    target.focus();
    target.setPointerCapture(event.pointerId);
    dragging = true;
    origin = orientation === "horizontal" ? event.clientY : event.clientX;
    initial = value;
  }
  function move(event: PointerEvent) {
    if (!dragging || !container) return;
    const extent =
      orientation === "horizontal"
        ? container.clientHeight
        : container.clientWidth;
    const position =
      orientation === "horizontal" ? event.clientY : event.clientX;
    if (extent > 0) update(initial - ((position - origin) / extent) * 100);
  }
  function key(event: KeyboardEvent) {
    const decrease = orientation === "horizontal" ? "ArrowDown" : "ArrowRight";
    const increase = orientation === "horizontal" ? "ArrowUp" : "ArrowLeft";
    if (![decrease, increase, "Home", "End"].includes(event.key)) return;
    event.preventDefault();
    update(
      event.key === "Home"
        ? min
        : event.key === "End"
          ? max
          : value + (event.key === increase ? 2 : -2),
    );
  }
</script>

<!-- A focusable, adjustable separator implements the ARIA window-splitter pattern. -->
<!-- svelte-ignore a11y_no_noninteractive_tabindex, a11y_no_noninteractive_element_interactions -->
<div
  role="separator"
  tabindex="0"
  aria-label={label}
  aria-orientation={orientation}
  aria-valuemin={min}
  aria-valuemax={max}
  aria-valuenow={Math.round(value)}
  class="layout-divider"
  class:horizontal={orientation === "horizontal"}
  class:dragging
  onpointerdown={start}
  onpointermove={move}
  onpointerup={() => (dragging = false)}
  onpointercancel={() => (dragging = false)}
  onlostpointercapture={() => (dragging = false)}
  onkeydown={key}
></div>

<style>
  .layout-divider {
    position: relative;
    z-index: 20;
    flex-shrink: 0;
    width: 1px;
    height: 100%;
    background: var(--hairline);
    cursor: col-resize;
    outline: none;
    touch-action: none;
  }
  .layout-divider::after {
    content: "";
    position: absolute;
    inset: 0 -4px;
  }
  .horizontal {
    width: 100%;
    height: 1px;
    cursor: row-resize;
  }
  .horizontal::after {
    inset: -4px 0;
  }
  .layout-divider:hover,
  .layout-divider:focus-visible,
  .dragging {
    background: var(--primary);
  }
  .layout-divider:hover::after,
  .layout-divider:focus-visible::after,
  .dragging::after {
    background: var(--primary);
    opacity: 0.25;
  }
</style>
