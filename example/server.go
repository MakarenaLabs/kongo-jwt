package main

import (
	"log"
	"net/http"

	"github.com/codegangsta/negroni"
	kong "github.com/giovanni-liboni/kongo-jwt"
	"github.com/gorilla/context"
	"github.com/gorilla/mux"
	"github.com/matryer/respond"
	"github.com/spf13/viper"
)

// TokenAuthentication contains the token released
type TokenAuthentication struct {
	Token string `json:"token" form:"token"`
}

// HandleAuthEndpoint retrive the autheticated Kong user with KongID, Username, ID (custom ID)
func HandleAuthEndpoint(w http.ResponseWriter, r *http.Request) {
	// Retrive auth user (if any)
	user := context.Get(r, "auth")
	respond.With(w, r, http.StatusOK, user)
}

// HandleLogin releases the token
func HandleLogin(w http.ResponseWriter, r *http.Request) {
	// Authenticate your users before call GetToken method with the username and the custom ID
	token, err := kong.GetToken("test", "123")
	if err != nil {
		respond.With(w, r, http.StatusInternalServerError, err.Error())
	} else {
		respond.With(w, r, http.StatusOK, TokenAuthentication{Token: token})
	}
	return
}

func main() {
	// User viper to configure the app
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		panic(err)
	}

	router := mux.NewRouter()
	router.HandleFunc("/endpoint_auth", HandleAuthEndpoint).Methods("GET")
	router.HandleFunc("/login", HandleLogin).Methods("POST")

	// Add the auth middleware to retrive the autheticated user at kong gateway
	n := negroni.New(kong.AuthMiddleware())
	n.UseHandler(router)

	log.Println("Web server started on port 8080")

	http.ListenAndServe(":8080", n)
}
