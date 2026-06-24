const INVALID_HANDLE = "handle.invalid";

export function isInvalidHandle(handle?: string | null): boolean {
  return !handle || handle === INVALID_HANDLE;
}

export function shortenDid(did?: string | null): string {
  if (!did) return "";
  if (did.startsWith("did:plc:")) {
    const id = did.slice("did:plc:".length);
    return id.length > 10 ? `did:plc:${id.slice(0, 4)}…${id.slice(-4)}` : did;
  }
  if (did.startsWith("did:web:")) {
    return did.slice("did:web:".length);
  }
  return did;
}

export function displayHandle(
  handle?: string | null,
  did?: string | null,
): string {
  if (isInvalidHandle(handle)) {
    return shortenDid(did) || handle || "";
  }
  return handle as string;
}
