// Validate S3 storage with synthetic bytes only; never opens an application DB.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/OdairPianta/unisinos_usp_api/storagebackend"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load(".env")
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	store, err := storagebackend.New(ctx)
	must(err)
	key := "files/validation/" + uuid.NewString() + ".txt"
	expected := []byte("Educação OnLIFE — validação sintética de armazenamento S3.\n")
	must(store.Put(ctx, key, expected, "text/plain; charset=utf-8", "validacao-onlife.txt"))
	defer func() { must(store.Delete(context.Background(), key)) }()
	signed, err := store.URL(ctx, key, "validacao-onlife.txt")
	must(err)
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(signed)
	must(err)
	data, err := io.ReadAll(io.LimitReader(resp.Body, 10240))
	resp.Body.Close()
	must(err)
	if resp.StatusCode != 200 || sha256.Sum256(data) != sha256.Sum256(expected) {
		panic("S3 content verification failed")
	}
	if !strings.HasPrefix(resp.Header.Get("Content-Disposition"), "attachment") {
		panic("Missing download disposition")
	}
	unsigned := strings.Split(signed, "?")[0]
	resp, err = client.Get(unsigned)
	must(err)
	resp.Body.Close()
	if resp.StatusCode != 403 {
		panic("Bucket is not private")
	}
	output, _ := json.Marshal(map[string]interface{}{"bucket": store.Bucket(), "signed_download": 200, "unsigned_download": 403, "sha256_matches": true, "credentials": "default AWS chain", "synthetic_object": key})
	fmt.Println(string(output))
}
func must(err error) {
	if err != nil {
		panic("S3 validation failed; credentials and signed URLs omitted")
	}
}
