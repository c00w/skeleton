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
}

func NewGateKeeper() (g *gatekeeper) {
	g = new(gatekeeper)
	g.objects = make(map[string]struct {
		value       string
		owner       string
		permissions map[string]bool
	})
	return g

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

func (g *gatekeeper) New(item, value, key string) (err error) {
	err = errors.New("Permission Denied")

	v, found := g.objects[item]

	if found {
		return
	}

	v.owner = key
	v.permissions = make(map[string]bool)
	v.permissions[key] = true
	v.value = value
	g.objects[item] = v
	return nil
}

func (g *gatekeeper) Set(item, value, key string) (err error) {
	err = errors.New("Permission Denied")

	v, found := g.objects[item]

	if found && v.owner != key {
		return
	}

	if !found {
		return
	}

	v.value = value
	g.objects[item] = v
	return nil
}

func (g *gatekeeper) Delete(item, key string) (err error) {
	err = errors.New("Permission Denied")

	v, found := g.objects[item]

	if found && v.owner != key {
		return
	}

	if !found {
		return
	}

	delete(g.objects, item)
	return nil
}

func (g *gatekeeper) AddAccess(item, key, newkey string) (err error) {
	err = errors.New("Permission Denied")

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

	v, found := g.objects[item]

	if found && v.owner != key {
		return
	}
	v.permissions[newkey] = false
	g.objects[item] = v
	return nil
}

func (g *gatekeeper) object(w http.ResponseWriter, r *http.Request) {
	key := r.FormValue("key")
	item := r.URL.Path
	s := strings.Split(item, "/")
	item = s[len(s)]

	var err error
	var v string

	value, err := ioutil.ReadAll(http.MaxBytesReader(w, r.Body, 1000000))

	switch r.Method {

	case "GET":
		v, err = g.Get(item, key)

	case "PUT":
		if err == nil {
			err = g.New(item, string(value), key)
		}

	case "POST":
		if err == nil {
			err = g.Set(item, string(value), key)
		}
	case "DELETE":
		if err == nil {
			err = g.Delete(item, key)
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

func (g *gatekeeper) permission(w http.ResponseWriter, r *http.Request) {
	key := r.FormValue("key")
	item := r.URL.Path
	s := strings.Split(item, "/")
	item = s[len(s)]

	var err error

	value, err := ioutil.ReadAll(http.MaxBytesReader(w, r.Body, 1000000))

	switch r.Method {

	case "POST":
		if err == nil {
			err = g.AddAccess(item, key, string(value))
		}

	case "DELETE":
		if err == nil {
			err = g.RemoveAccess(item, key, string(value))
		}
	}

	w.Header().Set("Content-Type", "text/plain; chaset=utf-8")
	if err != nil {
		w.WriteHeader(400)
		io.WriteString(w, err.Error())
		return
	}

	w.WriteHeader(200)
}

func main() {

	g := NewGateKeeper()

	http.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "gatekeeper v0")
	})

	http.HandleFunc("/object/", g.object)
	http.HandleFunc("/permissions/", g.permission)

	log.Print("starting gatekeeper")
	log.Fatal(http.ListenAndServe(":800", nil))
}
