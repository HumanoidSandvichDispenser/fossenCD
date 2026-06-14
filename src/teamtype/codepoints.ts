/**
 * Return the number of Unicode code points in `s`.
 * This is not the same as `s.length`, which counts UTF-16 code units, and will be larger than the
 * number of code points if `s` contains any characters outside the Basic Multilingual Plane (BMP).
 *
 * @param s The string to count code points in.
 * @returns The number of Unicode code points in `s`.
 */
export function codePointLength(s: string): number {
  let n = 0;
  for (let i = 0; i < s.length; ) {
    const c = s.codePointAt(i)!;
    i += c > 0xffff ? 2 : 1;
    n++;
  }
  return n;
}

/**
 * Advance `from` by `n` Unicode code points in `s`, and return the resulting index.
 *
 * @param s The string to advance through.
 * @param from The starting index in `s`.
 * @param n The number of Unicode code points to advance.
 * @returns The index in `s` that is `n` code points after `from`.
 */
export function advanceCodePoints(s: string, from: number, n: number): number {
  if (n <= 0) {
    return from;
  }
  let i = from;
  let cp = 0;
  while (cp < n && i < s.length) {
    const c = s.codePointAt(i)!;
    i += c > 0xffff ? 2 : 1;
    cp++;
  }
  return i;
}
