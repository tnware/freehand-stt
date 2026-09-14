// Use Vite’s normal Svelte resolution so the fixture shares the component runtime.
export { mount } from "svelte";
// Route singleton imports through Vite too, including its module-version URLs.
export { session } from "$lib/stores/session.svelte";
