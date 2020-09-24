package shortner

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

// Fault is interface for sending error message with code.
type Fault struct {
	Code      int
	Message   string
	Link      string
	ShortLink string
}

// HandleHomeLink Rendering the Home Page
func HandleHomeLink(response http.ResponseWriter, request *http.Request) {
	http.ServeFile(response, request, "views/index.html")
}

// HandlePostLink This function will return the response based on link found in Database
func HandlePostLink(w http.ResponseWriter, r *http.Request) {
	var req ShortLink
	var fault = Fault{
		Code: http.StatusInternalServerError, Message: "Something went wrong at our end",
	}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&req)

	if err != nil {
		fault.Code = http.StatusBadRequest
		fault.Message = "URL can't be empty"
		sendError(w, r, fault)
	} else if !IsValid(req.Original) {
		fault.Code = http.StatusBadRequest
		fault.Message = "An invalid URL found, provide a valid URL"
		sendError(w, r, fault)
	} else {
		createOrFetch(w, r, req)
	}
}

// HandleGetLink This function will redirect to actual website
func HandleGetLink(response http.ResponseWriter, request *http.Request) {
	var httpError = Fault{
		Code: http.StatusInternalServerError, Message: "Something went wrong at our end",
	}
	slug := mux.Vars(request)["slug"]
	if slug == "" {
		httpError.Code = http.StatusBadRequest
		httpError.Message = "URL Code can't be empty"
		sendError(response, request, httpError)
	} else {
		actual, err := Find(slug)
		if actual.Original == "" || err != nil {
			httpError.Code = http.StatusNotFound
			httpError.Message = "An invalid/expired URL Code found"
			sendError(response, request, httpError)
		} else {
			http.Redirect(response, request, actual.Original, http.StatusSeeOther)
		}
	}
}

func createOrFetch(w http.ResponseWriter, r *http.Request, req ShortLink) {
	var id string
	var err error
	var isNew bool
	q, e := FindByOriginalLink(req.Original)
	if e != nil {
		sendError(w, r, Fault{
			Code: http.StatusInternalServerError, Message: "Something went wrong at our end",
		})
	}
	if q.Expiry > 0 {
		id = q.Shortened
		err = nil
		isNew = false
	} else {
		isNew = true
		id, err = Shorten(req.Original)
	}
	if err != nil {
		sendError(w, r, Fault{
			Code: http.StatusInternalServerError, Message: "Something went wrong at our end",
		})
	} else {
		s := ShortLink{Shortened: id, Original: req.Original, Created: time.Now(), Hits: 0}
		sendLink(w, r, s, isNew)
	}
}

func sendLink(w http.ResponseWriter, r *http.Request, s ShortLink, isNew bool) {
	var fault Fault
	var obj ShortLink
	var err error
	if isNew {
		obj, err = Persist(s)
	} else {
		obj, err = s, nil
	}
	if err != nil {
		fault = Fault{Message: "Unable to save short link"}
		fmt.Println(err)
		sendError(w, r, fault)
	}

	fault = Fault{
		Code:      http.StatusOK,
		Message:   "Short URL generated",
		Link:      obj.Shortened,
		ShortLink: r.Host + "/" + obj.Shortened,
	}
	data, err := json.Marshal(fault)
	if err != nil {
		panic(err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(fault.Code)
	w.Write(data)
}

func sendError(w http.ResponseWriter, r *http.Request, m Fault) {
	response := &Fault{Code: m.Code, Message: m.Message}
	data, err := json.Marshal(response)
	if err != nil {
		panic(err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(m.Code)
	w.Write(data)
}
