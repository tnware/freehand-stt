/**
 * Display forms for the configured endpoint.
 *
 * The scheme is dropped from both: it is the same for every request and only
 * costs width in a row that has to hold a long host.
 */

const parse = (baseURL: string): URL | null => {
  try {
    return new URL(baseURL);
  } catch {
    return null;
  }
};

/** Host and path, for the main screen row: `speech.example.com/v1`. */
export const endpointLabel = (baseURL: string): string => {
  const url = parse(baseURL);
  if (!url) return baseURL;
  const path = url.pathname.replace(/\/+$/, "");
  return `${url.host}${path}`;
};

/** Host alone, for the status strip, where the path is noise. */
export const endpointHost = (baseURL: string): string => parse(baseURL)?.host ?? baseURL;
