package simple

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"io"
	"testing"
)

// 16,24,32位密钥
const (
	key     = "0123456789abcdef"
	content = `["127.0.0.1"]`
)

func TestAES_E(t *testing.T) {
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		t.Fatal(err)
	}

	ctx := pad(content)

	db := make([]byte, aes.BlockSize+len(ctx))
	iv := db[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		t.Fatal(err)
	}

	cipher.NewCBCEncrypter(block, iv).CryptBlocks(db[aes.BlockSize:], ctx)
	toBytes := IntToBytes(len(content))
	db = append(db, toBytes...)
	t.Log(hex.EncodeToString(db))
}

func TestAES_D(t *testing.T) {
	db, err := hex.DecodeString(content)
	if err != nil {
		t.Fatal(err)
	}

	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		t.Fatal(err)
	}

	iv := db[:aes.BlockSize]
	contentL := BytesToInt(db[len(db)-4:])
	ctx := db[aes.BlockSize:]
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(ctx, ctx[:len(ctx)-4])
	t.Log(string(ctx[:contentL]))
}

func pad(content string) (in []byte) {
	in = []byte(content)
	contentL := len(content)

	if remain := contentL % 16; remain != 0 {
		contentL = contentL + 16 - remain
		contentL = contentL - len(content)
		for i := 0; i < contentL; i++ {
			in = append(in, 0)
		}
	}
	return
}

func IntToBytes(n int) []byte {
	x := int32(n)
	buffer := bytes.NewBuffer([]byte{})
	_ = binary.Write(buffer, binary.BigEndian, x)
	return buffer.Bytes()
}

func BytesToInt(b []byte) int {
	buffer := bytes.NewBuffer(b)
	var x int32
	_ = binary.Read(buffer, binary.BigEndian, &x)
	return int(x)
}
