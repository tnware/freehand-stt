export async function hideThenToggle(hide: () => Promise<void>, toggle: () => Promise<void>) {
  await hide();
  await toggle();
}
