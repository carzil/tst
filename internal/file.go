package internal

import (
	"crypto/sha512"
	"encoding/hex"
	"io"
	"os"
)

const chunkSize = 4096

func FileChunkedChecksums(file *os.File) ([]string, error) {
	buf := make([]byte, chunkSize)
	pos := 0
	var chunks []string
	for {
		n, err := file.Read(buf)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		b := buf[:n]
		pos += n
		hash := sha512.New()
		_, _ = hash.Write(b)
		chunks = append(chunks, hex.EncodeToString(hash.Sum(nil)))
	}
	return chunks, nil
}
