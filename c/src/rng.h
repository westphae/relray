#ifndef RNG_H
#define RNG_H

#include <stdint.h>

// xoshiro256** PRNG — fast, high-quality, CUDA-compatible
typedef struct { uint64_t s[4]; } Rng;

static inline uint64_t rotl(const uint64_t x, int k) {
    return (x << k) | (x >> (64 - k));
}

static inline uint64_t rng_next(Rng *rng) {
    const uint64_t result = rotl(rng->s[1] * 5, 7) * 9;
    const uint64_t t = rng->s[1] << 17;
    rng->s[2] ^= rng->s[0];
    rng->s[3] ^= rng->s[1];
    rng->s[1] ^= rng->s[2];
    rng->s[0] ^= rng->s[3];
    rng->s[2] ^= t;
    rng->s[3] = rotl(rng->s[3], 45);
    return result;
}

// Returns a double in [0, 1)
static inline double rng_float64(Rng *rng) {
    return (double)(rng_next(rng) >> 11) * 0x1.0p-53;
}

// Seed from a single uint64
static inline Rng rng_seed(uint64_t seed) {
    Rng r;
    // SplitMix64 to expand seed into state
    for (int i = 0; i < 4; i++) {
        seed += 0x9e3779b97f4a7c15ULL;
        uint64_t z = seed;
        z = (z ^ (z >> 30)) * 0xbf58476d1ce4e5b9ULL;
        z = (z ^ (z >> 27)) * 0x94d049bb133111ebULL;
        r.s[i] = z ^ (z >> 31);
    }
    return r;
}

#endif
