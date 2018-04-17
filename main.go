package main

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"

	"golang.org/x/crypto/bcrypt"

	jwt "github.com/dgrijalva/jwt-go"
	scribble "github.com/nanobox-io/golang-scribble"
)

var t = InitTemplates()

func main() {
	log.SetFlags(log.Lshortfile)
	db, err := scribble.New("./db", nil)
	if err != nil {
		panic(err)
	}

	fs := http.FileServer(http.Dir("static"))
	http.Handle("/", fs)

	http.HandleFunc("/aboutUs.html", serveTemplate("aboutUs"))
	http.HandleFunc("/index.html", serveTemplate("index"))
	http.HandleFunc("/contactme.html", serveTemplate("contactme"))
	http.HandleFunc("/login_SignUp.html", serveTemplate("login_SignUp"))
	http.HandleFunc("/signup.html", serveTemplate("signup"))
	http.HandleFunc("/schedule.html", serveTemplate("schedule"))

	http.HandleFunc("/submitUser", func(w http.ResponseWriter, r *http.Request) {
		data, _ := ioutil.ReadAll(r.Body)
		q, err := url.ParseQuery(string(data))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		//checks if user has entered email already
		err = db.Read("users", q.Get("email"), nil)
		if err == nil {
			http.Error(w, "Email Taken", http.StatusConflict)
			log.Println(err)
			return
		}

		if q.Get("psw") != q.Get("psw-repeat") {
			http.Error(w, "Passwords dont match!!!", http.StatusConflict)
			return
		}

		q.Del("psw-repeat")
		pw := hashAndSalt([]byte(q.Get("psw")))
		if pw == "" {
			http.Error(w, "Password not long enough", http.StatusBadRequest)
			return
		}
		q.Set("psw", pw)

		err = db.Write("users", q.Get("email"), &q)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/login_SignUp.html", http.StatusTemporaryRedirect)
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
		if !comparePasswords(user.Get("psw"), []byte(q.Get("psw"))) {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		http.SetCookie(w, &http.Cookie{Name: "auth", Value: string(createJWT(q.Get("uname")))})
		http.Redirect(w, r, "/index.html", http.StatusTemporaryRedirect)
	})
	http.HandleFunc("/schedule", func(w http.ResponseWriter, r *http.Request) {

		c, err := r.Cookie("auth")
		if err != nil {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		u, valid := verifyJWT(c.Value)
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
		return ""
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
	token, err := jwt.Parse(tk, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("There was an error")
		}
		return []byte("secret"), nil
	})
	if err != nil {
		return "", false
	}
	claims, ok := token.Claims.(jwt.MapClaims)

	if !ok || !token.Valid {
		return "", false
	}
	return claims["username"].(string), true

}

func InitTemplates() *template.Template {
	files, _ := ioutil.ReadDir("./templates")
	fileNames := []string{}
	for _, f := range files {
		fileNames = append(fileNames, fmt.Sprint("./templates/", f.Name()))
	}

	s1, err := template.ParseFiles(fileNames...)
	if err != nil {
		panic(err)
	}
	return s1
}
func serveTemplate(s string) func(w http.ResponseWriter, r *http.Request) {

	return func(w http.ResponseWriter, r *http.Request) {

		t.ExecuteTemplate(w, s, getUser(r))

	}
}
func getUser(r *http.Request) string {
	c, err := r.Cookie("auth")
	if err != nil {
		log.Println(err)
		return "login/sign-up"
	}
	u, valid := verifyJWT(c.Value)
	if !valid {
		log.Println(err)
		return "login/sign-up"
	}
	return u
}
