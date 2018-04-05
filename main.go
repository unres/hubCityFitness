package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	scribble "github.com/nanobox-io/golang-scribble"
)

func main() {
	db, err := scribble.New("./db", nil)
	if err != nil {
		panic(err)
	}

	http.Handle("/", http.FileServer(http.Dir("static")))

	http.HandleFunc("/submitUser", func(w http.ResponseWriter, r *http.Request) {
		data, _ := ioutil.ReadAll(r.Body)
		q, err := url.ParseQuery(string(data))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		fmt.Println(string(data))
		fmt.Println(q)

		err = db.Write("users", q.Get("email"), &q)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		data, _ = json.MarshalIndent(&q, "", "    ")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write(data)
		http.StatusText(http.StatusOK)
	})

	http.ListenAndServe(":8080", nil)
}
