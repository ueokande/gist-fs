package main

import (
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
