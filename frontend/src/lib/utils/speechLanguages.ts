import { ID, type Profile } from "$bindings/modelprofile";
import type { VoicesResult } from "$bindings/inference";
import type { Option } from "$bindings/speechlanguage";

const baseLanguage = (language: string) => language.toLowerCase().split("-")[0];

export function speechLanguageOptions(
  profile: Profile | undefined,
  voices: VoicesResult | null,
): Option[] {
  const languages = profile?.languages ?? [];
  if (profile?.id !== ID.MagpieTTS) return languages;
  const advertised = !voices?.errorKind ? voices?.languages : undefined;
  return [
    { code: "", label: "Server default" },
    ...languages.filter((language) =>
      advertised?.length
        ? advertised.some(
            (code) => baseLanguage(code) === baseLanguage(language.code),
          )
        : !["zh", "ja"].includes(baseLanguage(language.code)),
    ),
  ];
}

export function speechLanguageValue(
  profile: Profile | undefined,
  language: string,
): string {
  if (profile?.id !== ID.MagpieTTS) return language || "auto";
  // Existing saved base aliases have the same qualified meaning; selecting a
  // different option is the only action that changes the saved value.
  return (
    profile.languages?.find(
      (option) =>
        option.code === language ||
        baseLanguage(option.code) === language.toLowerCase(),
    )?.code ?? language
  );
}
