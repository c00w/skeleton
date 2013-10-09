package libgatekeeper

import (
	"common"
	"errors"
	"io/ioutil"
	"log"
	"strings"
)

type Client struct {
	h   *common.HttpAPI
	key string
}

func NewOneTimeClient(address string, onetimekey string) (g *Client, err error) {
	g = NewClient(address, "")

	// Fetch the key & delete it
	key, err := g.Get("key." + onetimekey)
	if err != nil {
		return
	}

	err = g.Delete("key." + onetimekey)
	if err != nil {
		log.Print("error deleting onetime key")
	}
	g.key = key
	return
}

func NewClient(address string, key string) (g *Client) {
	g = &Client{common.NewHttpClient(address), key}
	return
}

func (g *Client) Get(key string) (value string, err error) {
	resp, err := g.h.Get("object/" + key + "?key=" + g.key)
	if err != nil {
		return
	}

	if resp.StatusCode != 200 {
		err = errors.New("Status code is " + resp.Status)
		return
	}
	c, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	value = string(c)
	return
}

func (g *Client) Set(item string, value string) (err error) {
	b := strings.NewReader(value)
	resp, err := g.h.Post("object/"+item+"?key="+g.key, "text/plain", b)
	if err != nil {
		return
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		return errors.New("Status code is " + resp.Status)
	}
	return
}

func (g *Client) New(item string, value string) (err error) {
	b := strings.NewReader(value)
	resp, err := g.h.Put("object/"+item+"?key="+g.key, "text/plain", b)
	if err != nil {
		return
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		return errors.New("Status code is " + resp.Status)
	}
	return
}

func (g *Client) Delete(item string) (err error) {
	resp, err := g.h.Delete("object/" + item + "?key=" + g.key)
	if err != nil {
		return
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		return errors.New("Status code is " + resp.Status)
	}
	return
}

func (g *Client) AddAccess(item string, newkey string) (err error) {
	b := strings.NewReader(newkey)
	resp, err := g.h.Post("permissions/"+item+"?key="+g.key, "text/plain", b)
	if err != nil {
		return
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		return errors.New("Status code is " + resp.Status)
	}
	return
}
