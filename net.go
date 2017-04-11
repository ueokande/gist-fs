package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
)

func getJson(url, username, password string, v interface{}) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(username, password)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	d := json.NewDecoder(resp.Body)

	if resp.StatusCode != http.StatusOK {
		var msg Error
		d.Decode(&msg)
		if err != nil {
			return err
		}
		return errors.New(msg.Message)
	}

	return d.Decode(v)
}

func patchJson(url, username, password string, body interface{}, v interface{}) error {
	input, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PATCH", url, bytes.NewBuffer(input))
	if err != nil {
		return err
	}
	req.SetBasicAuth(username, password)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	d := json.NewDecoder(resp.Body)

	if resp.StatusCode != http.StatusOK {
		var msg Error
		d.Decode(&msg)
		if err != nil {
			return err
		}
		return errors.New(msg.Message)
	}

	return d.Decode(v)
}
