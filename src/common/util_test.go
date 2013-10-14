package common

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"
)

func JsonTest(t *testing.T) {
	//write to buffer, write from buffer
	buff := new(bytes.Buffer)
	enc := NewEncWriter(buff)
	dec := json.NewDecoder(buff)
	m := &Message{}
	testString := "Throwing in a test string"
	enc.Log(testString)
	err := dec.Decode(m)
	if err != nil {
		t.Fatal("Encountered an error when trying to decode message string")
	}
	if m.Message != testString {
		t.Error("String was not passed correctly from encoder to decoder")
		t.Log("Expecting \"" + testString + "\"")
		t.Log("Received \"" + m.Message + "\"")
	}
	if m.Message_type != "message" {
		t.Error("Message type should be message, received " + m.Message_type)
	}
	if m.Status != "" {
		t.Error("Message should not be throwing a status, received " + m.Status)
	}
	errString := "Throwing in an error"
	enc.SetError(errors.New(errString))
	err = dec.Decode(m)
	if err != nil {
		t.Fatal("Encountered an error when trying to decode error string")
	}
	if m.Message != testString {
		t.Error("Error string was not passed correctly form encoder to decoder")
		t.Log("Expecting \"" + testString + "\"")
		t.Log("Received \"" + m.Message + "\"")
	}
	if m.Message_type != "error" {
		t.Error("Message type should be error, recieved " + m.Message_type)
	}
	if m.Status == "" {
		t.Error("Error messages should give a status, received no status.")
	}
}
