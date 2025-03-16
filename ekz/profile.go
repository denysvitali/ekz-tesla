package ekz

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type Profile struct {
	Personal struct {
		FirstName                string      `json:"first_name"`
		LastName                 string      `json:"last_name"`
		Email                    string      `json:"email"`
		Language                 string      `json:"language"`
		LanguageOfCorrespondence string      `json:"language_of_correspondence"`
		MobilePhone              interface{} `json:"mobile_phone"`
		Street                   string      `json:"street"`
		HouseNumber              string      `json:"house_number"`
		ZipCode                  string      `json:"zip_code"`
		City                     string      `json:"city"`
		Country                  string      `json:"country"`
		CompanyName              string      `json:"company_name"`
		IsSocialLogin            bool        `json:"is_social_login"`
		Telefone                 string      `json:"telefone"`
		UserId                   int         `json:"user_id"`
	} `json:"personal"`
}

func (c *Client) GetProfile() (*Profile, error) {
	res, err := c.httpClient.Get(Backend + "/users/profile")
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %s", res.Status)
	}

	var response Response[Profile]
	err = json.NewDecoder(res.Body).Decode(&response)
	if err != nil {
		return nil, err
	}
	return &response.Data, nil
}
