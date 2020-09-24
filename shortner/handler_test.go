package shortner

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gorilla/mux"
)

func TestHandlePostLink(t *testing.T) {
	form := newShortLinkForm()
	recorder := createPost(form, t)
	if status := recorder.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}
	s := ShortLink{}
	err := json.NewDecoder(recorder.Body).Decode(&s)
	if err != nil {
		t.Fatal(err)
	}
}

func TestHandlePostLinkWithAlreadyShortenedLink(t *testing.T) {
	form := newShortLinkForm()
	recorder := createPost(form, t)
	if status := recorder.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}
	fault := Fault{}
	err := json.NewDecoder(recorder.Body).Decode(&fault)
	if err != nil {
		t.Fatal(err)
	}
	shortlink := fault.Link
	recorder = createPost(form, t)
	if status := recorder.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}
	fault = Fault{}
	err = json.NewDecoder(recorder.Body).Decode(&fault)
	if err != nil {
		t.Fatal(err)
	}
	if shortlink != fault.Link {
		t.Errorf("wrong shortlink for existing url: got %v want %v",
			fault.Link, shortlink)
	}
}
func TestHandlePostLinkWithEmptyURL(t *testing.T) {
	form := make(map[string]string)
	form["url"] = ""
	recorder := createPost(form, t)
	if status := recorder.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusBadRequest)
	}
}

func TestHandlePostLinkWithInvalidURL(t *testing.T) {
	form := make(map[string]string)
	form["url"] = "htt.//www.zyx//"
	recorder := createPost(form, t)
	if status := recorder.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusBadRequest)
	}
}

func TestHandleHomeLink(t *testing.T) {

}

func TestHandleGetLink(t *testing.T) {
	form := newShortLinkForm()
	recorder := createPost(form, t)
	fault := Fault{}
	fmt.Println("response -> ", recorder.Body)
	err := json.NewDecoder(recorder.Body).Decode(&fault)

	fmt.Println("request -> ", "/"+fault.Link)
	req, err2 := http.NewRequest("GET", "/"+fault.Link, nil)
	if err != nil {
		t.Fatal(err2)
	}
	recorder = httptest.NewRecorder()
	r := mux.NewRouter().StrictSlash(true)
	r.HandleFunc("/{slug}", HandleGetLink).Methods("GET")
	http.Handle("/", r)
	r.ServeHTTP(recorder, req)

	if status := recorder.Code; status != http.StatusSeeOther {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusSeeOther)
	}
}
func TestHandleGetLinkWithUnknownShortLink(t *testing.T) {
	fmt.Println("request -> ", "/ZZZZ00000")
	req, err2 := http.NewRequest("GET", "/ZZZZ00000", nil)
	if err2 != nil {
		t.Fatal(err2)
	}
	recorder := httptest.NewRecorder()
	r := mux.NewRouter().StrictSlash(true)
	r.HandleFunc("/{slug}", HandleGetLink).Methods("GET")
	r.ServeHTTP(recorder, req)

	if status := recorder.Code; status != http.StatusNotFound {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusNotFound)
	}
}
func TestHandleGetLinkWithEmptySlug(t *testing.T) {
	fmt.Println("request -> ", "/")
	req, err2 := http.NewRequest("GET", "/", nil)
	if err2 != nil {
		t.Fatal(err2)
	}
	recorder := httptest.NewRecorder()
	r := mux.NewRouter().StrictSlash(true)
	r.HandleFunc("/{slug}", HandleGetLink).Methods("GET")
	r.ServeHTTP(recorder, req)

	if status := recorder.Code; status != http.StatusNotFound {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusNotFound)
	}
}

func newShortLinkForm() map[string]string {
	urls := []string{"https://www.youtube.com/show?", "http://www.stord.com/api/virtual/wharehouses"}
	longURL := urls[rand.Intn(2)] + GetRandom() + GetRandom()
	form := make(map[string]string)
	form["url"] = longURL
	return form
}

func createPost(form map[string]string, t *testing.T) *httptest.ResponseRecorder {
	body, _ := json.Marshal(form)
	fmt.Println("request -> ", string(body))
	req, err := http.NewRequest("POST", "/", bytes.NewReader(body))
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Content-Length", strconv.Itoa(len(body)))
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	r := mux.NewRouter()
	r.HandleFunc("/", HandlePostLink).Methods("POST")
	r.ServeHTTP(recorder, req)
	return recorder
}
