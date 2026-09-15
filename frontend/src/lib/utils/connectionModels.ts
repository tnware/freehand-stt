import {
  ConnectionProbe,
  ModelPresence,
  type ConnectionResult,
} from "$bindings/connection";
import { Purpose } from "$bindings/savedconnection";

/** A shared server inventory remains intact; each task gets its own choices. */
export function connectionModelsForPurpose(
  result: ConnectionResult,
  purpose: Purpose,
  selectedModel: string,
): ConnectionResult {
  if (!result.models?.length) return result;
  const capability =
    purpose === Purpose.Speech
      ? "speech"
      : purpose === Purpose.Cleanup
        ? "chat"
        : "transcription";
  const metadata = new Map(result.models.map((model) => [model.id, model]));
  const modelIDs =
    result.modelIDs?.filter((id) => {
      const advertised = metadata.get(id)?.capability;
      return !advertised || advertised === capability;
    }) ?? null;
  return {
    ...result,
    modelIDs,
    modelPresence:
      result.reachable &&
      result.probe === ConnectionProbe.ConnectionProbeModels &&
      selectedModel
        ? modelIDs?.includes(selectedModel)
          ? ModelPresence.ModelPresenceListed
          : ModelPresence.ModelPresenceNotListed
        : result.modelPresence,
  };
}
