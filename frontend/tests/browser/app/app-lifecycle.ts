import { mount, tick, unmount } from "svelte";
import App from "../../../src/App.svelte";
import { Session } from "$lib/stores/session.svelte";

// Exercise the real mount owner without starting native feature requests.
export async function mountLifecycleApp() {
  const session = new Session();
  let finishLoad!: () => void;
  session.load = () =>
    new Promise<void>((resolve) => {
      finishLoad = resolve;
    });
  let failures = 0;
  session.messages.fail = () => {
    failures++;
  };
  session.messages.reportFailure = () => {
    failures++;
  };
  const target = document.createElement("div");
  document.body.append(target);
  const app = mount(App, { target, props: { session } });
  await tick();
  return {
    finishLoad: async () => {
      finishLoad();
      await tick();
    },
    unmount: async () => {
      await unmount(app);
      target.remove();
    },
    failures: () => failures,
  };
}
