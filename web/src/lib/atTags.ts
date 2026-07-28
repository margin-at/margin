export interface AtTags {
  canonical?: string[];
  alternate?: string[];
  author?: string[];
  me?: string[];
}

const AT_URI_RE =
  /^at:\/\/did:[a-z0-9]+:[A-Za-z0-9._:%-]+(\/[A-Za-z0-9.-]+(\/[A-Za-z0-9._~:@!$&'()*+,;=%-]+)?)?$/;

const AT_DID_URI_RE = /^at:\/\/did:[a-z0-9]+:[A-Za-z0-9._:%-]+$/;

export function isAtUri(value: unknown): value is string {
  return typeof value === "string" && AT_URI_RE.test(value);
}

export function isAtDidUri(value: unknown): value is string {
  return typeof value === "string" && AT_DID_URI_RE.test(value);
}

export function atUriForDid(
  did: string | null | undefined,
): string | undefined {
  if (!did || !did.startsWith("did:")) return undefined;
  const uri = `at://${did}`;
  return isAtDidUri(uri) ? uri : undefined;
}

function sanitize(
  values: (string | undefined)[] | undefined,
  validate: (value: unknown) => boolean,
): string[] {
  if (!values) return [];
  return [
    ...new Set(values.filter((value): value is string => validate(value))),
  ];
}

export function normalizeAtTags(tags: AtTags): Required<AtTags> {
  return {
    canonical: sanitize(tags.canonical, isAtUri),
    alternate: sanitize(tags.alternate, isAtUri),
    author: sanitize(tags.author, isAtDidUri),
    me: sanitize(tags.me, isAtDidUri),
  };
}
