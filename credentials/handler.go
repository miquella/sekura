package credentials

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/awserr"
)

const (
	handlerExpirationFormat = "2006-01-02T15:04:05Z"
)

type Handler interface {
	http.Handler
	GetAuthToken() string
}

func NewHandler(provider aws.CredentialsProvider) (Handler, error) {
	authToken := make([]byte, 120, 120)
	_, err := rand.Read(authToken)
	if err != nil {
		return nil, err
	}

	return &handler{
		authToken: base64.RawURLEncoding.EncodeToString(authToken),
		provider:  provider,
	}, nil
}

type handlerMessage struct {
	AccessKeyID     string `json:"AccessKeyId,omitempty"`
	SecretAccessKey string `json:"SecretAccessKey,omitempty"`
	Token           string `json:"Token,omitempty"`
	Expiration      string `json:"Expiration,omitempty"`

	ErrorCode    string `json:"code,omitempty"`
	ErrorMessage string `json:"message,omitempty"`
}

type handler struct {
	authToken string
	provider  aws.CredentialsProvider
}

func (h *handler) GetAuthToken() string {
	return h.authToken
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// validate the method
	if r.Method != "GET" {
		w.Header().Set("Allow", "GET")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// consume the body so the transport can be reused
	io.Copy(ioutil.Discard, r.Body)
	r.Body.Close()

	// validate the request and return credentials
	status := http.StatusOK
	message := handlerMessage{}

	if h.validAuthToken(r) {
		creds, err := h.provider.Retrieve()
		if err == nil {
			message.AccessKeyID = creds.AccessKeyID
			message.SecretAccessKey = creds.SecretAccessKey
			message.Token = creds.SessionToken

			if creds.CanExpire {
				message.Expiration = creds.Expires.Format(handlerExpirationFormat)

				if creds.Expires.Before(time.Now()) {
					status = http.StatusGone
					message.ErrorCode = "CredentialsExpired"
					message.ErrorMessage = "Credentials have expired and cannot be renewed"

					log.Printf("WARNING: Credentials have expired and cannot be renewed")
				}
			} else {
				// TODO: FIND A BETTER WAY TO HANDLE THIS
				message.Expiration = time.Now().Add(15 * time.Second).Format(handlerExpirationFormat)
			}
		} else {
			status = http.StatusInternalServerError
			if aerr, ok := err.(awserr.Error); ok {
				message.ErrorCode = aerr.Code()
				message.ErrorMessage = aerr.Message()
			} else {
				message.ErrorCode = "CredentialsUnavailable"
				message.ErrorMessage = "Credentials cannot be retrieved"
			}

			log.Printf("ERROR: Failed to retreive creentials: %v", err)
		}
	} else {
		status = http.StatusForbidden
		message.ErrorCode = "InvalidAuthorizationToken"
		message.ErrorMessage = "The authorization token is invalid"
	}

	// marshal the body
	body, err := json.MarshalIndent(message, "", "  ")
	if err != nil {
		log.Printf("ERROR: Failed to marshal JSON message: %v", err)
		http.Error(w, "Failed to marshal JSON message", http.StatusInternalServerError)
		return
	}

	// write the header and body
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	w.WriteHeader(status)

	_, err = w.Write(body)
	if err != nil {
		log.Printf("ERROR: Failed to write body: %v", err)
	}
}

// BE VERY CAREFUL MODIFYING THIS METHOD!
//
// Seemingly innocuous modifications could be detrimental here.
//
// This method is very sensitive and should not be allowed to leak any
// unnecessary information.
func (h *handler) validAuthToken(r *http.Request) bool {
	if h.authToken == "" {
		return true
	}

	// TODO: CHANGE THIS TO NORMALIZE THE LENGTH TO THE LENGTH OF THE USER TOKEN,
	//       THEN COMPARE THE STRINGS, THEN COMPARE THE LENGTHS

	// not sure that we can accomplish a constant-time comparision of tokens if
	// they are different lengths, so we are knowingly leaking information
	// about the true length of the token
	userAuthToken := r.Header.Get("Authorization")
	return subtle.ConstantTimeCompare([]byte(h.authToken), []byte(userAuthToken)) == 1
}
