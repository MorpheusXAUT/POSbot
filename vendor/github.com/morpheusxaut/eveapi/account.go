package eveapi

const (
	AccountStatusURL     = "/account/AccountStatus.xml.aspx"
	AccountAPIKeyInfoURL = "/account/APIKeyInfo.xml.aspx"
	//AccountCharactersURL is a duplicate of the above call
	//AccountCharactersURL = "/account/Characters.xml.aspx"
)

//AccountAPIKeyInfo fetches info this key (such as characters attached)
func (api API) AccountAPIKeyInfo() (*APIKeyInfoResponse, error) {
	output := APIKeyInfoResponse{}
	err := api.Call(AccountAPIKeyInfoURL, nil, &output)
	if err != nil {
		return nil, err
	}

	if output.Error != nil {
		return nil, output.Error
	}

	return &output, nil
}

//APIKeyInfoResponse details the api key in use
type APIKeyInfoResponse struct {
	APIResult
	Key APIKey `xml:"result>key"`
}

//APIKey the api key being used
type APIKey struct {
	AccessMask int             `xml:"accessMask,attr"`
	Type       string          `xml:"type,attr"`
	Rows       []APIKeyInfoRow `xml:"rowset>row"`
}

//APIKeyInfoRow details the characters the api key is for
type APIKeyInfoRow struct {
	ID              int    `xml:"characterID,attr"`
	Name            string `xml:"characterName,attr"`
	CorporationID   int    `xml:"corporationID,attr"`
	CorporationName string `xml:"corporationName,attr"`
	AllianceID      int    `xml:"allianceID,attr"`
	AllianceName    string `xml:"allianceName,attr"`
	FactionID       int    `xml:"factionID,attr"`
	FactionName     string `xml:"factionName,attr"`
}

//AccountStatus fetches info on the status of the account
func (api API) AccountStatus() (*AccountStatusResponse, error) {
	output := AccountStatusResponse{}
	err := api.Call(AccountStatusURL, nil, &output)
	if err != nil {
		return nil, err
	}

	if output.Error != nil {
		return nil, output.Error
	}

	return &output, nil
}

type AccountStatusResponse struct {
	APIResult
	PaidUntil    eveTime `xml:"result>paidUntil"`
	CreateDate   eveTime `xml:"result>createDate"`
	LogonCount   int     `xml:"result>logonCount"`
	LogonMinutes int     `xml:"result>logonMinutes"`
}

func (api API) AccountGetChars() ([]string, error) {
	output := make([]string, 0)
	apikeyinfo, err := api.AccountAPIKeyInfo()
	if err != nil {
		return nil, err
	}
	for _, chars := range apikeyinfo.Key.Rows {
		output = append(output, chars.Name)
	}
	return output, nil
}
