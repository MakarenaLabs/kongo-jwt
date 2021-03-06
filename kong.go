package kongojwt

// Sezione per la comunicazione con Kong, mette a disposizione delle API per
// interagire con il gateway
import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/spf13/viper"
)

// JSON returned from new user request
type KongCustomer struct {
	CreatedAt int    `json:"created_at"`
	CustomID  string `json:"custom_id"`
	ID        string `json:"id"`
	Username  string `json:"username"`
}

// JSON ritornato quando si crea il JWT
type JWTResult struct {
	Algorithm  string `json:"algorithm"`
	ConsumerID string `json:"consumer_id"`
	CreatedAt  int    `json:"created_at"`
	ID         string `json:"id"`
	Key        string `json:"key"`
	Secret     string `json:"secret"`
}

type JWTResults struct {
	Data  []JWTResult `json:"data"`
	Total int         `json:"total"`
}

type KongData struct {
	Customer   KongCustomer
	JWTResult  JWTResult
	JWTResults JWTResults
	Username   string
	CustomID   string
	Token      string
}

func (data *KongData) CreateCustomer() error {
	r, err := http.PostForm(viper.GetString("kong_server")+"/consumers", url.Values{"username": {data.Username}, "custom_id": {data.CustomID}})
	if err != nil {
		return err
	} else {
		if r.StatusCode == http.StatusCreated {
			decoder := json.NewDecoder(r.Body)
			err := decoder.Decode(&data.Customer)
			if err != nil {
				return err
			}
			return nil
		} else if r.StatusCode == http.StatusConflict {
			// User already exists
			return nil
		}
		// Code not handled before
		return errors.New(r.Status)
	}
}

func (data *KongData) CreateJWTCredentials() error {
	r, err := http.PostForm(viper.GetString("kong_server")+"/consumers/"+data.Username+"/jwt", url.Values{})
	if err != nil {
		return err
	}
	if r.StatusCode == http.StatusCreated {
		decoder := json.NewDecoder(r.Body)
		err := decoder.Decode(&data.JWTResult)
		if err != nil {
			return err
		}
		// Create token from new credentials
		err = data.GenerateToken()
		if err != nil {
			return err
		}
		return nil
	}
	return nil
}

func (data *KongData) GetJWTCredentials() (int, error) {
	r, err := http.Get(viper.GetString("kong_server") + "/consumers/" + data.Username + "/jwt")
	if err != nil {
		return http.StatusInternalServerError, err
	}

	if r.StatusCode == http.StatusNotFound {
		// User not found, return nil as error
		return http.StatusNotFound, nil
	} else if r.StatusCode == http.StatusFound || r.StatusCode == http.StatusOK {
		decoder := json.NewDecoder(r.Body)
		err := decoder.Decode(&data.JWTResults)
		if err != nil {
			return http.StatusInternalServerError, err
		}
		log.Println(data.JWTResults.Total)
		return http.StatusFound, nil
	}
	return r.StatusCode, errors.New(r.Status)
}

func (data *KongData) SetDefaultJWTResult() error {
	if data.JWTResults.Total > 0 {
		// Select first result
		data.JWTResult = data.JWTResults.Data[0]
		err := data.GenerateToken()
		if err != nil {
			return err
		}
		return nil
	}
	return nil
}

func (data *KongData) GenerateToken() error {
	var err error
	token := jwt.New(jwt.SigningMethodHS256)
	token.Claims["exp"] = time.Now().Add(time.Minute * time.Duration(60)).Unix()
	token.Claims["iat"] = time.Now().Unix()
	token.Claims["sub"] = data.CustomID
	token.Claims["iss"] = data.JWTResult.Key
	data.Token, err = token.SignedString([]byte(data.JWTResult.Secret))
	if err != nil {
		return err
	}
	return nil
}
