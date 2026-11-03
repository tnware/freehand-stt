import { AuthenticationMode } from "$bindings/config";
import { ID, Role } from "$bindings/compatibility";
import {
  Purpose,
  type Change,
  type Connection,
} from "$bindings/savedconnection";
import type { Instance, ProviderDescriptor } from "$bindings/managedruntime";
import type { SettingsDTO } from "$bindings/settings";

/** Opt-in UI fixture for backend-owned runtime connections, never live inference. */
export function publishRuntimeConnections(
  settings: SettingsDTO,
  instances: Instance[],
  providers: ProviderDescriptor[],
) {
  const entries = settings.savedConnections.entries ?? [];
  const generated: Connection[] = instances.map((instance) => {
    const provider = providers.find((entry) => entry.id === instance.provider);
    const primary = provider?.models?.find(
      (model) => model.id === instance.model,
    );
    const speech = provider?.models?.find(
      (model) => model.id === (instance.speechModel || instance.model),
    );
    const transcribes = primary?.contracts?.some(
      (contract) => contract.role === Role.Transcription,
    );
    const prior = entries.find(
      (entry) =>
        entry.builtIn && entry.details.managedInstanceID === instance.id,
    );
    return {
      id: prior?.id ?? `builtin-fixture-${instance.id}`,
      name: instance.name,
      builtIn: true,
      hasCredential: false,
      uses: [
        ...(transcribes ? [Purpose.Voice, Purpose.Transcription] : []),
        ...(primary?.contracts?.some(
          (contract) => contract.role === Role.PostProcessing,
        )
          ? [Purpose.Cleanup]
          : []),
        ...(speech?.contracts?.some((contract) => contract.role === Role.Speech)
          ? [Purpose.Speech]
          : []),
      ],
      details: {
        managedInstanceID: instance.id,
        compatibilityProfile: ID.$zero,
        baseURL: "",
        allowInsecureHTTP: false,
        authenticationMode: AuthenticationMode.AuthenticationModeNone,
        healthPath: "",
        headers: {},
      },
    };
  });
  settings.savedConnections = {
    selected: { ...settings.savedConnections.selected },
    entries: [...entries.filter((entry) => !entry.builtIn), ...generated],
  };
}

export function selectRuntimeConnection(
  settings: SettingsDTO,
  change: Change,
  providers: ProviderDescriptor[],
) {
  const purpose = change.purpose;
  const connection = settings.savedConnections.entries?.find(
    (entry) => entry.id === change.id,
  );
  if (!purpose || !connection?.builtIn || !connection.uses?.includes(purpose))
    throw new Error("Fixture runtime connection cannot serve this workflow");
  const instance = settings.managedRuntimes?.find(
    (row) => row.id === connection.details.managedInstanceID,
  );
  const role = purpose === Purpose.Speech ? Role.Speech : Role.Transcription;
  const modelID =
    purpose === Purpose.Speech
      ? instance?.speechModel || instance?.model
      : instance?.model;
  const model = providers
    .find((provider) => provider.id === instance?.provider)
    ?.models?.find((model) => model.id === modelID);
  const contract = model?.contracts?.find((contract) => contract.role === role);
  if (!instance || !model || !contract)
    throw new Error("Fixture selected model is unavailable");
  const endpoint =
    purpose === Purpose.Voice
      ? settings.voiceTranscription
      : purpose === Purpose.Transcription
        ? settings
        : settings.textToSpeech;
  Object.assign(endpoint, connection.details, {
    model: model.id,
    compatibilityProfile: contract.compatibilityProfile,
    modelProfile: contract.modelProfile,
  });
  settings.savedConnections.selected = {
    ...settings.savedConnections.selected,
    [purpose]: connection.id,
  };
}
