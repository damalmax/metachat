package skype

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"math/big"
)

func skypeHMACSHA256(input, id, key string) string {
	inputInts := []uint8(input)
	idInts := []uint8(id)
	keyInts := []uint8(key)

	paddingLen := (8 - ((len(inputInts) + len(idInts)) % 8)) % 8
	padding := make([]uint8, paddingLen)
	for i := range padding {
		padding[i] = '0'
	}

	message8 := append(inputInts, idInts...)
	message8 = append(message8, padding...)

	message32 := to32(message8)

	sha := sha256.Sum256(append(inputInts, keyInts...))
	truncSha := sha[:16]
	sha256Parts := to32(truncSha)

	maxInt32 := big.NewInt(0x7fffffff)
	magic := big.NewInt(0x0e79a9c1)
	hash0 := new(big.Int).And(big.NewInt(int64(sha256Parts[0])), maxInt32)
	hash1 := new(big.Int).And(big.NewInt(int64(sha256Parts[1])), maxInt32)
	hash2 := new(big.Int).And(big.NewInt(int64(sha256Parts[2])), maxInt32)
	hash3 := new(big.Int).And(big.NewInt(int64(sha256Parts[3])), maxInt32)
	temp := big.NewInt(0)
	low := big.NewInt(0)
	high := big.NewInt(0)

	for i := 0; i <= len(message32)-2; i = i + 2 {
		message := big.NewInt(int64(message32[i+1]))
		temp = temp.Mul(message, magic).Mod(temp, maxInt32)
		low = low.Add(low, temp).Mul(low, hash0).Add(low, hash1).Mod(low, maxInt32)
		high = high.Add(high, low)

		temp = message
		low = low.Add(low, temp).Mul(low, hash2).Add(low, hash3).Mod(low, maxInt32)
		high = high.Add(high, low)
	}

	var checkSum64 []uint32
	checkSum64 = append(checkSum64, uint32(low.Add(low, hash1).Mod(low, maxInt32).Uint64()))
	checkSum64 = append(checkSum64, uint32(high.Add(high, hash3).Mod(high, maxInt32).Uint64()))

	var output32 []uint32
	output32 = append(output32, sha256Parts[0]^checkSum64[0])
	output32 = append(output32, sha256Parts[1]^checkSum64[1])
	output32 = append(output32, sha256Parts[2]^checkSum64[0])
	output32 = append(output32, sha256Parts[3]^checkSum64[1])

	output8 := make([]uint8, 0)
	for _, v := range output32 {
		buf := make([]uint8, 4)
		binary.LittleEndian.PutUint32(buf, v)
		output8 = append(output8, buf...)
	}

	return hex.EncodeToString(output8)
}

func to32(input []uint8) []uint32 {
	var buf []uint8
	var result []uint32
	for _, v := range input {
		if len(buf) < 4 {
			buf = append(buf, v)
		}

		if len(buf) == 4 {
			result = append(result, binary.LittleEndian.Uint32(buf))
			buf = nil
		}
	}

	return result
}
