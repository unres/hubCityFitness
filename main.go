package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"

	"golang.org/x/crypto/bcrypt"

	jwt "github.com/dgrijalva/jwt-go"
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

		//checks if user has entered email already
		_, err = os.Stat("./db/users/" + q.Get("email") + ".json")
		if err == nil {
			http.Error(w, "Email Taken", http.StatusConflict)
			return
		}
		fmt.Println(err)
		if !os.IsNotExist(err) {
			http.Error(w, "Email Taken", http.StatusConflict)
			return
		}

		if q.Get("psw") != q.Get("psw-repeat") {
			http.Error(w, "Passwords dont match!!!", http.StatusConflict)
			return
		}

		q.Del("psw-repeat")
		pw := hashAndSalt([]byte(q.Get("psw")))
		q.Set("psw", pw)

		err = db.Write("users", q.Get("email"), &q)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		fmt.Fprintf(w, "User: %s signed up", q.Get("email"))

		http.StatusText(http.StatusOK)
	})

	http.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		data, _ := ioutil.ReadAll(r.Body)
		q, err := url.ParseQuery(string(data))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		var user url.Values

		err = db.Read("users", q.Get("uname"), &user)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if q.Get("uname") != user.Get("email") {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		if comparePasswords(q.Get("psw"), []byte(user.Get("psw"))) {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		w.Write(createJWT(q.Get("uname")))
	})
	http.HandleFunc("/schedule", func(w http.ResponseWriter, r *http.Request) {

		u, valid := verifyJWT(r.Header.Get("Authorization"))
		if !valid {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		fmt.Fprintf(w, "hello %s", u)
	})

	http.ListenAndServe(":8080", nil)
}

func hashAndSalt(pwd []byte) string {

	// Use GenerateFromPassword to hash & salt pwd.
	// MinCost is just an integer constant provided by the bcrypt
	// package along with DefaultCost & MaxCost.
	// The cost can be any value you want provided it isn't lower
	// than the MinCost (4)
	hash, err := bcrypt.GenerateFromPassword(pwd, bcrypt.MinCost)
	if err != nil {
		log.Println(err)
	}

	// GenerateFromPassword returns a byte slice so we need to
	// convert the bytes to a string and return it
	return string(hash)
}

func comparePasswords(hashedPwd string, plainPwd []byte) bool {

	// Since we'll be getting the hashed password from the DB it
	// will be a string so we'll need to convert it to a byte slice
	byteHash := []byte(hashedPwd)

	err := bcrypt.CompareHashAndPassword(byteHash, plainPwd)
	if err != nil {
		log.Println(err)
		return false
	}

	return true

}

func createJWT(username string) []byte {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": username,
	})
	tokenString, err := token.SignedString([]byte("secret"))
	if err != nil {
		fmt.Println(err)
		return nil
	}
	return []byte(tokenString)
}

func verifyJWT(tk string) (string, bool) {
	token, _ := jwt.Parse(tk, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("There was an error")
		}
		return []byte("secret"), nil
	})
	claims, ok := token.Claims.(jwt.MapClaims)

	if !ok || !token.Valid {
		return "", false
	}
	return claims["username"].(string), true

}
