package main

import (
	"errors"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

type gatekeeper struct {
	objects map[string]struct {
		value       string
		owner       string
		permissions map[string]bool
	}

	owners map[string]bool
}

func (g *gatekeeper) Get(item, key string) (value string, err error) {
	err = errors.New("No Such Item or Permission Denied")

	o, ok := g.objects[item]
	if !ok {
		return
	}

	ok = o.permissions[key]
	if !ok {
		return
	}

	return o.value, nil
}

func (g *gatekeeper) Set(item, value, key string) (err error) {
	err = errors.New("Permission Denied")

	ok := g.owners[key]
	if !ok {
		return
	}

	v, found := g.objects[item]

	if found && v.owner != key {
		return
	}

	if !found {
		v.owner = key
		v.permissions[key] = true
	}

	v.value = value
	g.objects[item] = v
	return nil
}

func (g *gatekeeper) AddAccess(item, key, newkey string) (err error) {
	err = errors.New("Permission Denied")

	ok := g.owners[key]
	if !ok {
		return
	}

	v, found := g.objects[item]

	if found && v.owner != key {
		return
	}

	v.permissions[newkey] = true
	g.objects[item] = v
	return nil
}

func (g *gatekeeper) RemoveAccess(item, key, newkey string) (err error) {
	err = errors.New("Permission Denied")

	ok := g.owners[key]
	if !ok {
		return
	}

	v, found := g.objects[item]

	if found && v.owner != key {
		return
	}
	v.permissions[newkey] = false
	g.objects[item] = v
	return nil
}

func (g *gatekeeper) object(w http.ResponseWriter, r *http.Request) {
	//Extract information from query
	key := r.FormValue("key")
	item := r.URL.Path
	s := strings.Split(item, "/")
	item = s[len(s)]

	var err error
	var v string

	switch r.Method {

	case "GET":
		v, err = g.Get(item, key)

	case "PUT":
	case "POST":
		value, err := ioutil.ReadAll(r.Body)
		if err == nil {
			err = g.Set(item, string(value), key)
		}
	}

	w.Header().Set("Content-Type", "text/plain; chaset=utf-8")
	if err != nil {
		w.WriteHeader(400)
		io.WriteString(w, err.Error())
		return
	}

	w.WriteHeader(200)
	io.WriteString(w, v)
}

func main() {

	g := new(gatekeeper)

	http.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "gatekeeper v0")
	})

	http.HandleFunc("/object/", g.object)
	http.HandleFunc("/permissions/", g.object)

	log.Fatal(http.ListenAndServe(":800", nil))
}
