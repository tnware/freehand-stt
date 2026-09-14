import { Purpose, type Connection } from "$bindings/savedconnection";
import { Role } from "$bindings/compatibility";
import type {
  Instance,
  InstanceStatus,
  ProviderDescriptor,
} from "$bindings/managedruntime";
import type { SettingsSectionID } from "$lib/navigation";

export const connectionWorkflows: {
  id: Purpose;
  label: string;
  section: SettingsSectionID;
}[] = [
  {
    id: Purpose.Voice,
    label: "Voice transcription",
    section: "voice-transcription",
  },
  {
    id: Purpose.Transcription,
    label: "Audio-file transcription",
    section: "server",
  },
  { id: Purpose.Cleanup, label: "Cleanup", section: "processing" },
  { id: Purpose.Speech, label: "Text to speech", section: "speech" },
];
export function managedConnectionSupports(
  instance: Instance | undefined,
  providers: ProviderDescriptor[],
  purpose: Purpose,
): boolean {
  const role =
    purpose === Purpose.Cleanup
      ? Role.PostProcessing
      : purpose === Purpose.Speech
        ? Role.Speech
        : Role.Transcription;
  const model = providers
    .find((p) => p.id === instance?.provider)
    ?.models?.find((m) => m.id === instance?.model);
  return !!model?.contracts?.some((contract) => contract.role === role);
}
export function connectionProvider(
  connection: Connection | undefined,
  instances: InstanceStatus[] = [],
): string | undefined {
  const id = connection?.details.managedInstanceID;
  return id
    ? instances.find((entry) => entry.instance.id === id)?.instance.provider
    : connection?.details.compatibilityProfile;
}
export function connectionTargetLabel(
  connection: Connection,
  instances: InstanceStatus[] = [],
): string {
  const id = connection.details.managedInstanceID;
  if (!id) return connection.details.baseURL;
  const row = instances.find((entry) => entry.instance.id === id);
  return `Local · ${row?.instance.name ?? id} · ${row?.status.state === "running" ? "Running" : row?.status.state || "Unavailable"}`;
}
export function connectionMatches(
  connection: Connection,
  query: string,
  instances: InstanceStatus[] = [],
): boolean {
  const words = query.trim().toLocaleLowerCase().split(/\s+/);
  const row = instances.find(
    (entry) => entry.instance.id === connection.details.managedInstanceID,
  );
  const text =
    `${connection.name} ${connectionTargetLabel(connection, instances)} ${connection.details.compatibilityProfile} ${row?.instance.provider ?? ""} ${row?.instance.model ?? ""} ${connection.details.managedInstanceID ? "managed local runtime instance offline" : "manual server"}`.toLocaleLowerCase();
  return words.every((word) => text.includes(word));
}
export function connectionSection(purpose: Purpose): SettingsSectionID {
  return (
    connectionWorkflows.find((workflow) => workflow.id === purpose)?.section ??
    "connections"
  );
}
