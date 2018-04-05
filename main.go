package main

import (
	"encoding/json"
	"net/http"

	scribble "github.com/nanobox-io/golang-scribble"
)

func main() {
	db, err := scribble.New("./db", nil)
	if err != nil {
		panic(err)
	}

	http.HandleFunc("/submitUser", func(w http.ResponseWriter, r *http.Request) {
		obj := map[string]interface{}{}
		json.NewDecoder(r.Body).Decode(&obj)
		db.Write("t", "1", &obj)
	})

	http.ListenAndServe(":8080", nil)
}
