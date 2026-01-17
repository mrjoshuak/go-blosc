//go:build arm64

#include "textflag.h"

// Shuffle mask for typeSize=4: rearranges bytes within a 128-bit vector
// Input:  [a0 a1 a2 a3 | b0 b1 b2 b3 | c0 c1 c2 c3 | d0 d1 d2 d3]
// Output: [a0 b0 c0 d0 | a1 b1 c1 d1 | a2 b2 c2 d2 | a3 b3 c3 d3]
//
// VTBL index values (each byte position gets its value from the index):
// Position 0:  get byte 0 (a0)   -> index 0x00
// Position 1:  get byte 4 (b0)   -> index 0x04
// Position 2:  get byte 8 (c0)   -> index 0x08
// Position 3:  get byte 12 (d0)  -> index 0x0c
// Position 4:  get byte 1 (a1)   -> index 0x01
// Position 5:  get byte 5 (b1)   -> index 0x05
// Position 6:  get byte 9 (c1)   -> index 0x09
// Position 7:  get byte 13 (d1)  -> index 0x0d
// Position 8:  get byte 2 (a2)   -> index 0x02
// Position 9:  get byte 6 (b2)   -> index 0x06
// Position 10: get byte 10 (c2)  -> index 0x0a
// Position 11: get byte 14 (d2)  -> index 0x0e
// Position 12: get byte 3 (a3)   -> index 0x03
// Position 13: get byte 7 (b3)   -> index 0x07
// Position 14: get byte 11 (c3)  -> index 0x0b
// Position 15: get byte 15 (d3)  -> index 0x0f
// Low 8 bytes (positions 0-7): [00 04 08 0c 01 05 09 0d]
// High 8 bytes (positions 8-15): [02 06 0a 0e 03 07 0b 0f]
DATA shuffle4_tbl<>+0(SB)/8, $0x0d0905010c080400
DATA shuffle4_tbl<>+8(SB)/8, $0x0f0b07030e0a0602
GLOBL shuffle4_tbl<>(SB), RODATA, $16

// Unshuffle mask for typeSize=4: reverse the shuffle
// Input:  [a0 b0 c0 d0 | a1 b1 c1 d1 | a2 b2 c2 d2 | a3 b3 c3 d3]
// Output: [a0 a1 a2 a3 | b0 b1 b2 b3 | c0 c1 c2 c3 | d0 d1 d2 d3]
//
// VTBL index values (each byte position gets its value from the index):
// Position 0:  get byte 0 (a0)   -> index 0x00
// Position 1:  get byte 4 (a1)   -> index 0x04
// Position 2:  get byte 8 (a2)   -> index 0x08
// Position 3:  get byte 12 (a3)  -> index 0x0c
// Position 4:  get byte 1 (b0)   -> index 0x01
// Position 5:  get byte 5 (b1)   -> index 0x05
// Position 6:  get byte 9 (b2)   -> index 0x09
// Position 7:  get byte 13 (b3)  -> index 0x0d
// Position 8:  get byte 2 (c0)   -> index 0x02
// Position 9:  get byte 6 (c1)   -> index 0x06
// Position 10: get byte 10 (c2)  -> index 0x0a
// Position 11: get byte 14 (c3)  -> index 0x0e
// Position 12: get byte 3 (d0)   -> index 0x03
// Position 13: get byte 7 (d1)   -> index 0x07
// Position 14: get byte 11 (d2)  -> index 0x0b
// Position 15: get byte 15 (d3)  -> index 0x0f
// Low 8 bytes (positions 0-7): [00 04 08 0c 01 05 09 0d]
// High 8 bytes (positions 8-15): [02 06 0a 0e 03 07 0b 0f]
DATA unshuffle4_tbl<>+0(SB)/8, $0x0d0905010c080400
DATA unshuffle4_tbl<>+8(SB)/8, $0x0f0b07030e0a0602
GLOBL unshuffle4_tbl<>(SB), RODATA, $16

// func shuffleBytesNEON(dst, src []byte, typeSize int) bool
// Arguments:
//   dst: slice at 0(FP), 24 bytes (ptr, len, cap)
//   src: slice at 24(FP), 24 bytes (ptr, len, cap)
//   typeSize: int at 48(FP), 8 bytes
//   ret: bool at 56(FP), 1 byte
TEXT ·shuffleBytesNEON(SB), NOSPLIT, $0-57
    MOVD    dst_base+0(FP), R0      // dst pointer
    MOVD    dst_len+8(FP), R1       // dst length (n)
    MOVD    src_base+24(FP), R2     // src pointer
    MOVD    src_len+32(FP), R3      // src length
    MOVD    typeSize+48(FP), R4     // typeSize

    // Check if we can use NEON: need typeSize == 4 and at least 16 bytes
    CMP     $4, R4
    BNE     shuffle_fallback
    CMP     $16, R1
    BLT     shuffle_fallback

    // Calculate number of elements and number of 16-byte chunks
    LSR     $2, R1, R5              // numElements = n / 4
    LSR     $2, R5, R6              // numChunks = numElements / 4
    CBZ     R6, shuffle_fallback    // If no full chunks, fallback

    // For shuffle output, we need to write to 4 separate regions:
    // - bytes 0 of each element go to dst[0:numElements]
    // - bytes 1 of each element go to dst[numElements:2*numElements]
    // - bytes 2 of each element go to dst[2*numElements:3*numElements]
    // - bytes 3 of each element go to dst[3*numElements:4*numElements]

    // Load shuffle table into V16
    MOVD    $shuffle4_tbl<>(SB), R7
    VLD1    (R7), [V16.B16]

    // Calculate destination offsets
    // R8 = dst + 0 (byte 0 output)
    // R9 = dst + numElements (byte 1 output)
    // R10 = dst + 2*numElements (byte 2 output)
    // R11 = dst + 3*numElements (byte 3 output)
    MOVD    R0, R8
    ADD     R5, R0, R9
    ADD     R5, R9, R10
    ADD     R5, R10, R11

    MOVD    $0, R12                 // chunk counter

shuffle_loop:
    // Load 16 bytes (4 elements)
    VLD1    (R2), [V0.B16]

    // Shuffle using table lookup
    // V0 contains: [a0 a1 a2 a3 | b0 b1 b2 b3 | c0 c1 c2 c3 | d0 d1 d2 d3]
    // After VTBL:  [a0 b0 c0 d0 | a1 b1 c1 d1 | a2 b2 c2 d2 | a3 b3 c3 d3]
    VTBL    V16.B16, [V0.B16], V1.B16

    // Extract and store each 4-byte group to its destination
    // Byte position 0: bytes 0-3 of V1
    VMOV    V1.S[0], R13
    MOVW    R13, (R8)

    // Byte position 1: bytes 4-7 of V1
    VMOV    V1.S[1], R13
    MOVW    R13, (R9)

    // Byte position 2: bytes 8-11 of V1
    VMOV    V1.S[2], R13
    MOVW    R13, (R10)

    // Byte position 3: bytes 12-15 of V1
    VMOV    V1.S[3], R13
    MOVW    R13, (R11)

    // Advance pointers
    ADD     $16, R2, R2             // src += 16
    ADD     $4, R8, R8              // dst byte 0 region += 4
    ADD     $4, R9, R9              // dst byte 1 region += 4
    ADD     $4, R10, R10            // dst byte 2 region += 4
    ADD     $4, R11, R11            // dst byte 3 region += 4

    ADD     $1, R12, R12
    CMP     R6, R12
    BLT     shuffle_loop

    MOVD    $1, R0
    MOVB    R0, ret+56(FP)
    RET

shuffle_fallback:
    MOVD    $0, R0
    MOVB    R0, ret+56(FP)
    RET

// func unshuffleBytesNEON(dst, src []byte, typeSize int) bool
TEXT ·unshuffleBytesNEON(SB), NOSPLIT, $0-57
    MOVD    dst_base+0(FP), R0      // dst pointer
    MOVD    dst_len+8(FP), R1       // dst length (n)
    MOVD    src_base+24(FP), R2     // src pointer
    MOVD    src_len+32(FP), R3      // src length
    MOVD    typeSize+48(FP), R4     // typeSize

    // Check if we can use NEON: need typeSize == 4 and at least 16 bytes
    CMP     $4, R4
    BNE     unshuffle_fallback
    CMP     $16, R1
    BLT     unshuffle_fallback

    // Calculate number of elements and number of 16-byte chunks
    LSR     $2, R1, R5              // numElements = n / 4
    LSR     $2, R5, R6              // numChunks = numElements / 4
    CBZ     R6, unshuffle_fallback  // If no full chunks, fallback

    // Load unshuffle table into V16
    MOVD    $unshuffle4_tbl<>(SB), R7
    VLD1    (R7), [V16.B16]

    // Calculate source offsets (reading from scattered locations)
    // R8 = src + 0 (byte 0 input)
    // R9 = src + numElements (byte 1 input)
    // R10 = src + 2*numElements (byte 2 input)
    // R11 = src + 3*numElements (byte 3 input)
    MOVD    R2, R8
    ADD     R5, R2, R9
    ADD     R5, R9, R10
    ADD     R5, R10, R11

    MOVD    $0, R12                 // chunk counter

unshuffle_loop:
    // Load 4 bytes from each byte-position region into a vector
    MOVWU   (R8), R13               // bytes 0: a0 b0 c0 d0
    MOVWU   (R9), R14               // bytes 1: a1 b1 c1 d1
    MOVWU   (R10), R15              // bytes 2: a2 b2 c2 d2
    MOVWU   (R11), R16              // bytes 3: a3 b3 c3 d3

    // Build V0: [a0 b0 c0 d0 | a1 b1 c1 d1 | a2 b2 c2 d2 | a3 b3 c3 d3]
    VMOV    R13, V0.S[0]
    VMOV    R14, V0.S[1]
    VMOV    R15, V0.S[2]
    VMOV    R16, V0.S[3]

    // Unshuffle using table lookup
    // This reverses the shuffle pattern
    VTBL    V16.B16, [V0.B16], V1.B16

    // Store 16 bytes (4 elements in original order)
    VST1    [V1.B16], (R0)

    // Advance pointers
    ADD     $16, R0, R0             // dst += 16
    ADD     $4, R8, R8              // src byte 0 region += 4
    ADD     $4, R9, R9              // src byte 1 region += 4
    ADD     $4, R10, R10            // src byte 2 region += 4
    ADD     $4, R11, R11            // src byte 3 region += 4

    ADD     $1, R12, R12
    CMP     R6, R12
    BLT     unshuffle_loop

    MOVD    $1, R0
    MOVB    R0, ret+56(FP)
    RET

unshuffle_fallback:
    MOVD    $0, R0
    MOVB    R0, ret+56(FP)
    RET
