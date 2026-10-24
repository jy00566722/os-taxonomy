// getRandomValues is also available over HTTP; randomUUID is secure-context only.
export function newRequestKey(source: Crypto = globalThis.crypto): string {
  if (typeof source.randomUUID === 'function') return source.randomUUID();
  return Array.from(source.getRandomValues(new Uint8Array(16)), b => b.toString(16).padStart(2, '0')).join('');
}
