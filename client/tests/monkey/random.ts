// Deterministic pseudo-random number generation for the monkey test.
//
// `Math.random()` cannot be seeded, so rerunning "with the same seed" would
// not actually reproduce the same sequence of actions. This module hashes an
// arbitrary seed string down to a 32-bit integer and feeds it into a small,
// fast PRNG (mulberry32) whose entire output is determined by that integer.

/** Hashes an arbitrary string into a 32-bit unsigned integer (FNV-1a). */
function hashSeed(seed: string): number {
  let h = 0x811c9dc5; // FNV-1a offset basis
  for (let i = 0; i < seed.length; i++) {
    h ^= seed.charCodeAt(i);
    h = Math.imul(h, 0x01000193); // FNV-1a prime
  }
  return h >>> 0;
}

/** mulberry32: a tiny, fast, deterministic PRNG. Given the same seed it
 *  always produces the same sequence of numbers in [0, 1). */
function mulberry32(seed: number): () => number {
  let a = seed >>> 0;
  return () => {
    a = (a + 0x6d2b79f5) | 0;
    let t = Math.imul(a ^ (a >>> 15), 1 | a);
    t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t;
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
}

/**
 * A deterministic random generator with a handful of convenience helpers.
 * Every random decision the monkey test makes goes through an instance of
 * this class, so the entire run is reproducible from a single seed string.
 */
export class SeededRandom {
  private readonly next: () => number;

  constructor(public readonly seed: string) {
    this.next = mulberry32(hashSeed(seed));
  }

  /** Random float in [0, 1). */
  float(): number {
    return this.next();
  }

  /** Random integer in [min, max], inclusive on both ends. */
  int(min: number, max: number): number {
    return Math.floor(this.float() * (max - min + 1)) + min;
  }

  /** Returns true with probability `p` (0..1). */
  chance(p: number): boolean {
    return this.float() < p;
  }

  /** Picks a uniformly random element from a non-empty array. */
  pick<T>(items: readonly T[]): T {
    if (items.length === 0) {
      throw new Error("SeededRandom.pick: cannot pick from an empty array");
    }
    return items[this.int(0, items.length - 1)];
  }

  /** Generates a random printable string, `minLen`–`maxLen` characters long. */
  string(minLen = 3, maxLen = 12): string {
    const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789 !@#$%";
    const len = this.int(minLen, maxLen);
    let out = "";
    for (let i = 0; i < len; i++) out += alphabet[this.int(0, alphabet.length - 1)];
    return out;
  }
}

/**
 * Resolves the seed to use for this run: `MONKEY_SEED` if set, otherwise a
 * freshly generated one. Always meant to be printed at the start of the test
 * so a failing run can be replayed exactly via `MONKEY_SEED=<seed>`.
 */
export function resolveSeed(): string {
  const fromEnv = process.env.MONKEY_SEED;
  if (fromEnv && fromEnv.trim() !== "") return fromEnv.trim();
  // Not cryptographic — this just needs to vary from run to run.
  return `auto-${Date.now()}-${Math.floor(Math.random() * 1_000_000)}`;
}
