package drive

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"io"
	"math/big"
	"slices"
)

// reference: https://github.com/kkHAIKE/fake115/blob/master/fake115d.user.js

var (
	kts = []byte{
		0xf0, 0xe5, 0x69, 0xae, 0xbf, 0xdc, 0xbf, 0x8a,
		0x1a, 0x45, 0xe8, 0xbe, 0x7d, 0xa6, 0x73, 0xb8,
		0xde, 0x8f, 0xe7, 0xc4, 0x45, 0xda, 0x86, 0xc4,
		0x9b, 0x64, 0x8b, 0x14, 0x6a, 0xb4, 0xf1, 0xaa,
		0x38, 0x01, 0x35, 0x9e, 0x26, 0x69, 0x2c, 0x86,
		0x00, 0x6b, 0x4f, 0xa5, 0x36, 0x34, 0x62, 0xa6,
		0x2a, 0x96, 0x68, 0x18, 0xf2, 0x4a, 0xfd, 0xbd,
		0x6b, 0x97, 0x8f, 0x4d, 0x8f, 0x89, 0x13, 0xb7,
		0x6c, 0x8e, 0x93, 0xed, 0x0e, 0x0d, 0x48, 0x3e,
		0xd7, 0x2f, 0x88, 0xd8, 0xfe, 0xfe, 0x7e, 0x86,
		0x50, 0x95, 0x4f, 0xd1, 0xeb, 0x83, 0x26, 0x34,
		0xdb, 0x66, 0x7b, 0x9c, 0x7e, 0x9d, 0x7a, 0x81,
		0x32, 0xea, 0xb6, 0x33, 0xde, 0x3a, 0xa9, 0x59,
		0x34, 0x66, 0x3b, 0xaa, 0xba, 0x81, 0x60, 0x48,
		0xb9, 0xd5, 0x81, 0x9c, 0xf8, 0x6c, 0x84, 0x77,
		0xff, 0x54, 0x78, 0x26, 0x5f, 0xbe, 0xe8, 0x1e,
		0x36, 0x9f, 0x34, 0x80, 0x5c, 0x45, 0x2c, 0x9b,
		0x76, 0xd5, 0x1b, 0x8f, 0xcc, 0xc3, 0xb8, 0xf5,
	}

	keyL = []byte{
		0x78, 0x06, 0xad, 0x4c, 0x33, 0x86, 0x5d, 0x18,
		0x4c, 0x01, 0x3f, 0x46,
	}

	rsaPk *rsa.PublicKey
)

func init() {
	n, _ := big.NewInt(0).SetString(
		"8686980c0f5a24c4b9d43020cd2c22703ff3f450756529058b1cf88f09b86021"+
			"36477198a6e2683149659bd122c33592fdb5ad47944ad1ea4d36c6b172aad633"+
			"8c3bb6ac6227502d010993ac967d1aef00f0c8e038de2e4d3bc2ec368af2e9f1"+
			"0a6f1eda4f7262f136420c07c331b871bf139f74f3010e3c4fe57df3afb71683", 16)
	e, _ := big.NewInt(0).SetString("10001", 16)
	rsaPk = &rsa.PublicKey{N: n, E: int(e.Int64())}
}

type Key [16]byte

func EncryptKey() Key {
	key := Key{}
	_, _ = io.ReadFull(rand.Reader, key[:])
	return key
}

func Encrypt(msg []byte, key Key) (output string) {
	buf := make([]byte, 16+len(msg))
	copy(buf, key[:])
	copy(buf[16:], msg)
	xorEncode(buf[16:], xorKey(key[:], 4))
	slices.Reverse(buf[16:])
	xorEncode(buf[16:], keyL)
	output = base64.StdEncoding.EncodeToString(rsaEncrypt(buf))
	return
}

func Decrypt(msg string, key Key) ([]byte, error) {
	data, err := base64.StdEncoding.DecodeString(msg)
	if err != nil {
		return nil, err
	}
	data = rsaDecrypt(data)
	buf := make([]byte, len(data)-16)
	copy(buf, data[16:])
	xorEncode(buf, xorKey(data[:16], 12))
	slices.Reverse(buf)
	xorEncode(buf, xorKey(key[:], 4))
	return buf, nil
}

func xorKey(seed []byte, size int) []byte {
	key := make([]byte, size)
	for i := 0; i < size; i++ {
		key[i] = (seed[i] + kts[size*i]) & 0xff
		key[i] ^= kts[size*(size-i-1)]
	}
	return key
}

func xorEncode(data []byte, key []byte) {
	dataSize, keySize := len(data), len(key)
	mod := dataSize % 4
	if mod > 0 {
		for i := 0; i < mod; i++ {
			data[i] ^= key[i%keySize]
		}
	}
	for i := mod; i < dataSize; i++ {
		data[i] ^= key[(i-mod)%keySize]
	}
}

// encrypt data use rsa public key
func rsaEncrypt(input []byte) []byte {
	inputSize, blockSize := len(input), rsaPk.Size()-11
	output := bytes.Buffer{}
	for index := 0; index < inputSize; index += blockSize {
		chunkSize := blockSize
		if index+chunkSize > inputSize {
			chunkSize = inputSize - index
		}
		p, err := rsa.EncryptPKCS1v15(rand.Reader, rsaPk, input[index:index+chunkSize])
		if err == nil {
			output.Write(p)
		}
	}
	return output.Bytes()
}

// decrypt data use rsa public key
func rsaDecrypt(input []byte) []byte {
	var output []byte
	inputSize, blockSize := len(input), rsaPk.Size()
	for index := 0; index < inputSize; index += blockSize {
		chunkSize := blockSize
		if index+chunkSize > inputSize {
			chunkSize = inputSize - index
		}
		n := big.NewInt(0).SetBytes(input[index : index+chunkSize])
		m := big.NewInt(0).Exp(n, big.NewInt(int64(rsaPk.E)), rsaPk.N)
		p := m.Bytes()
		i := bytes.IndexByte(p, '\x00')
		if i < 0 {
			return nil
		}
		output = append(output, p[i+1:]...)
	}
	return output
}
