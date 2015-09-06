package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"

	"gopkg.in/mgo.v2"

	log "github.com/Sirupsen/logrus"
)

const salt = "[replace this with something unique]"

const maxSnippetSize = 64 * 1024

// Snippet is type of order to save the code made with playground.
type Snippet struct {
	ID   string
	Body []byte
}

func (s *Snippet) setID() string {
	h := sha1.New()
	io.WriteString(h, salt)
	h.Write(s.Body)
	sum := h.Sum(nil)
	b := make([]byte, base64.URLEncoding.EncodedLen(len(sum)))
	base64.URLEncoding.Encode(b, sum)

	id := string(b)[:10]
	s.ID = id
	return id
}

func init() {
	http.HandleFunc("/share", share)
}

func share(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	var body bytes.Buffer
	_, err := io.Copy(&body, io.LimitReader(r.Body, maxSnippetSize+1))
	r.Body.Close()
	if err != nil {
		log.Errorf("reading Body: %v", err)
		http.Error(w, "Server Error", http.StatusInternalServerError)
		return
	}
	if body.Len() > maxSnippetSize {
		http.Error(w, "Snippet is too large", http.StatusRequestEntityTooLarge)
		return
	}

	snip := &Snippet{Body: body.Bytes()}
	id := snip.setID()

	// mongo
	session, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}
	defer session.Close()
	session.SetMode(mgo.Monotonic, true)
	c := session.DB("playground").C("snippet")
	err = c.Insert(snip)
	if err != nil {
		log.Errorf("putting Snippet: %v", err)
		http.Error(w, "Server Error", http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, id)
}