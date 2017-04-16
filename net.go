package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

type RestClient struct {
	Username string
	Password string
}

func (c *RestClient) newRequest(method, url string, params interface{}) (*http.Request, error) {
	var body io.Reader
	if params != nil {
		input, err := json.Marshal(params)
		if err != nil {
			return nil, err
		}
		body = bytes.NewBuffer(input)
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(c.Username, c.Password)
	return req, nil
}

func (c *RestClient) Do(method, url string, params interface{}, result interface{}) error {
	req, err := c.newRequest(method, url, params)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	dec := json.NewDecoder(resp.Body)
	if resp.StatusCode != http.StatusOK {
		var msg Error
		dec.Decode(&msg)
		if err != nil {
			return err
		}
		return errors.New(msg.Message)
	}
	if result == nil {
		return nil
	}
	return dec.Decode(result)
}

func (c *RestClient) Get(url string, result interface{}) error {
	return c.Do(http.MethodGet, url, nil, result)
}

func (c *RestClient) Patch(url string, params interface{}, result interface{}) error {
	return c.Do(http.MethodPatch, url, params, result)
}

func (c *RestClient) Post(url string, params interface{}, result interface{}) error {
	return c.Do(http.MethodPost, url, params, result)
}

func (c *RestClient) Put(url string, params interface{}, result interface{}) error {
	return c.Do(http.MethodPut, url, params, result)
}

func (c *RestClient) Delete(url string, params interface{}, result interface{}) error {
	return c.Do(http.MethodDelete, url, params, result)
}
