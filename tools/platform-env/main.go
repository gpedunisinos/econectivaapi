// Read dotenv with exactly the same parser as the API. The maintenance caller
// captures stdout privately; never invoke it in a public pipeline log.
package main

import (
	"encoding/json"
	"github.com/joho/godotenv"
	"os"
)

func main() {
	if len(os.Args) != 2 {
		os.Exit(2)
	}
	v, e := godotenv.Read(os.Args[1])
	if e != nil {
		os.Exit(1)
	}
	json.NewEncoder(os.Stdout).Encode(v)
}
