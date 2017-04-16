package main

import "encoding/json"

func (f *EditGistForm) MarshalJSON() ([]byte, error) {
	hash := make(map[string]interface{})
	if f.Description != nil {
		hash["description"] = f.Description
	}
	files := make(map[string]interface{})
	for k, v := range f.Files {
		if v == nil {
			files[k] = nil
		} else {
			file := make(map[string]interface{})
			if v.Filename != nil {
				file["filename"] = v.Filename
			}
			if v.Content != nil {
				file["content"] = v.Content
			}
			files[k] = file
		}
	}
	hash["files"] = files
	return json.Marshal(hash)
}
