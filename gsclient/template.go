package gsclient

import (
	"log"
)

type Templates struct {
	List map[string]TemplateProperties `json:"sshkeys"`
}

type Template struct {
	Properties TemplateProperties `json:"sshkey"`
}

type TemplateProperties struct {
	Name       string `json:"name"`
	ObjectUuid string `json:"object_uuid"`
	Status     string `json:"status"`
	CreateTime string `json:"create_time"`
	ChangeTime string `json:"change_time"`
	Template   string `json:"sshkey"`
}

type TemplateCreateRequest struct {
	Name         string   `json:"name"`
	Labels       []string `json:"labels"`
	SnapshotUuid string   `json:"name"`
}

func (c *Client) GetTemplate(id string) (*Template, error) {
	r := Request{
		uri:    "/objects/templates/" + id,
		method: "GET",
	}

	response := new(Template)
	err := r.execute(*c, &response)

	log.Printf("Received sshkey: %v", response)

	return response, err
}

func (c *Client) GetTemplateList() (*Templates, error) {
	r := Request{
		uri:    "/objects/templates/",
		method: "GET",
	}

	response := new(Templates)
	err := r.execute(*c, &response)

	log.Printf("Received template: %v", response)

	return response, err
}

func (c *Client) CreateTemplate(body TemplateCreateRequest) (*CreateResponse, error) {
	r := Request{
		uri:    "/objects/templates",
		method: "POST",
		body:   body,
	}

	response := new(CreateResponse)
	err := r.execute(*c, &response)
	if err != nil {
		return nil, err
	}

	err = c.WaitForRequestCompletion(response.RequestUuid)

	return response, err
}

func (c *Client) DeleteTemplate(id string) error {
	r := Request{
		uri:    "/objects/template/" + id,
		method: "DELETE",
	}

	return r.execute(*c, nil)
}

func (c *Client) UpdateTemplate(id string, body map[string]interface{}) error {
	r := Request{
		uri:    "/objects/sshkeys/" + id,
		method: "PATCH",
		body:   body,
	}

	return r.execute(*c, nil)
}