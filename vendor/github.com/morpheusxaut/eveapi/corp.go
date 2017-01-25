package eveapi

import (
	"net/url"
	"strconv"
)

const (
	CorpContactListURL     = "/corp/ContactList.xml.aspx"
	CorpAccountBalanceURL  = "/corp/AccountBalance.xml.aspx"
	CorpStarbaseListURL    = "/corp/StarbaseList.xml.aspx"
	CorpStarbaseDetailsURL = "/corp/StarbaseDetail.xml.aspx"
	CorpWalletJournalURL   = "/corp/WalletJournal.xml.aspx"
)

type Contact struct {
	ID       string `xml:"contactID,attr"`
	Name     string `xml:"contactName,attr"`
	Standing int    `xml:"standing,attr"`
}

type ContactSubList struct {
	Name     string    `xml:"name,attr"`
	Contacts []Contact `xml:"row"`
}

func (api API) CorpContactList() (*ContactList, error) {
	output := ContactList{}
	err := api.Call(CorpContactListURL, nil, &output)
	if err != nil {
		return nil, err
	}
	return &output, nil
}

type ContactList struct {
	APIResult
	ContactList []ContactSubList `xml:"result>rowset"`
}

func (c ContactList) Corporate() []Contact {
	for _, v := range c.ContactList {
		if v.Name == "corporateContactList" {
			return v.Contacts
		}
	}
	return nil
}
func (c ContactList) Alliance() []Contact {
	for _, v := range c.ContactList {
		if v.Name == "allianceContactList" {
			return v.Contacts
		}
	}
	return nil
}

type AccountBalance struct {
	APIResult
	Accounts []struct {
		ID      int     `xml:"accountID,attr"`
		Key     int     `xml:"accountKey,attr"`
		Balance float64 `xml:"balance,attr"`
	} `xml:"result>rowset>row"`
}

func (api API) CorpAccountBalances() (*AccountBalance, error) {
	output := AccountBalance{}
	err := api.Call(CorpAccountBalanceURL, nil, &output)
	if err != nil {
		return nil, err
	}
	if output.Error != nil {
		return nil, output.Error
	}
	return &output, nil
}

type StarbaseList struct {
	APIResult
	Starbases []*Starbase `xml:"result>rowset>row"`
}

type Starbase struct {
	ID              int           `xml:"itemID,attr"`
	TypeID          int           `xml:"typeID,attr"`
	LocationID      int           `xml:"locationID,attr"`
	MoonID          int           `xml:"moonID,attr"`
	State           StarbaseState `xml:"state,attr"`
	StateTimestamp  eveTime       `xml:"stateTimestamp,attr"`
	OnlineTimestamp eveTime       `xml:"onlineTimestamp,attr"`
	StandingOwnerID int           `xml:"standingOwnerID,attr"`
}

type StarbaseState int

const (
	StarbaseStateUnanchored StarbaseState = 0
	StarbaseStateAnchored   StarbaseState = 1
	StarbaseStateOnlining   StarbaseState = 2
	StarbaseStateReinforced StarbaseState = 3
	StarbaseStateOnline     StarbaseState = 4
)

func (s StarbaseState) String() string {
	switch s {
	case StarbaseStateUnanchored:
		return "unanchored"
	case StarbaseStateAnchored:
		return "anchored/offline"
	case StarbaseStateOnlining:
		return "onlining"
	case StarbaseStateReinforced:
		return "reinforced"
	case StarbaseStateOnline:
		return "online"
	default:
		return "unknown/invalid"
	}
}

func (api API) CorpStarbaseList() (*StarbaseList, error) {
	output := StarbaseList{}
	err := api.Call(CorpStarbaseListURL, nil, &output)
	if err != nil {
		return nil, err
	}
	if output.Error != nil {
		return nil, output.Error
	}
	return &output, nil
}

type StarbaseDetails struct {
	APIResult
	State           StarbaseState `xml:"result>state"`
	StateTimestamp  eveTime       `xml:"result>stateTimestamp"`
	OnlineTimestamp eveTime       `xml:"result>onlineTimestamp"`
	GeneralSettings struct {
		UsageFlags              int  `xml:"usageFlags"`
		DeployFlags             int  `xml:"deployFlags"`
		AllowCorporationMembers bool `xml:"allowCorporationMembers"`
		AllowAllianceMembers    bool `xml:"allowAllianceMembers"`
	} `xml:"result>generalSettings"`
	CombatSettings struct {
		UseStandingsFrom struct {
			OwnerID int `xml:"ownerID,attr"`
		} `xml:"useStandingsFrom"`
		OnStandingDrop struct {
			Standing int `xml:"standing,attr"`
		} `xml:"onStandingDrop"`
		OnStatusDrop struct {
			Enabled  bool `xml:"enabled,attr"`
			Standing int  `xml:"standing,attr"`
		} `xml:"onStatusDrop"`
		OnAgression struct {
			Enabled bool `xml:"enabled,attr"`
		} `xml:"onAgression"`
		OnCorporationWar struct {
			Enabled bool `xml:"enabled, attr"`
		} `xml:"onCorporationWar"`
	} `xml:"result>combatSettings"`
	Fuel []StarbaseFuel `xml:"result>rowset>row"`
}

type StarbaseFuel struct {
	TypeID   int `xml:"typeID,attr"`
	Quantity int `xml:"quantity,attr"`
}

func (api API) CorpStarbaseDetails(starbaseID int) (*StarbaseDetails, error) {
	output := StarbaseDetails{}
	args := url.Values{}
	args.Set("itemID", strconv.Itoa(starbaseID))
	err := api.Call(CorpStarbaseDetailsURL, args, &output)
	if err != nil {
		return nil, err
	}
	if output.Error != nil {
		return nil, output.Error
	}
	return &output, nil
}

//MarketOrder is either a sell order or buy order
type CorpWalletJournal struct {
	TransactionDateTime eveTime `xml:"date,attr"`       //datetime  Date and time of transaction.
	RefID               int64   `xml:"refID,attr"`      //long      Unique journal reference ID.
	RefTypeID           int     `xml:"refTypeID,attr"`  //int       Transaction type.
	OwnerName1          string  `xml:"ownerName1,attr"` //string    Name of first party in transaction.
	OwnerID1            int64   `xml:"ownerID1,attr"`   //long Character or corporation ID of first party. For NPC corporations, see the appropriate cross reference.
	OwnerName2          string  `xml:"ownerName2,attr"` //string    Name of second party in transaction.
	OwnerID2            int64   `xml:"ownerID2,attr"`   //long Character or corporation ID of second party. For NPC corporations, see the appropriate cross reference.
	ArgName1            string  `xml:"argName1,attr"`   //string    Ref type dependent argument name.
	ArgID1              int     `xml:"argID1,attr"`     //int 	Ref type dependent argument value.
	Amount              float64 `xml:"amount,attr"`     //decimal   Transaction amount. Positive when value transferred to the first owner. Negative otherwise.
	Balance             float64 `xml:"balance,attr"`    //decimal   Wallet balance after transaction occurred.
	Reason              string  `xml:"reason,attr"`     //string    	Ref type dependent reason.

	// OwnerTypes:
	// 2 = Corporation
	// 1373-1386 = Character
	// 16159 = Alliance
	Owner1TypeID int `xml:"owner1TypeID,attr"` //int 		Determines the owner type.
	Owner2TypeID int `xml:"owner2TypeID,attr"` //int 		Determines the owner type.
}

type CorpWalletJournalResult struct {
	APIResult
	Transactions []CorpWalletJournal `xml:"result>rowset>row"`
}

//WalletTransactions returns the wallet journal for the current corp
//accountKey	int	Account key of the wallet for which transactions will be returned. Corporations have seven wallets with accountKeys numbered from 1000 through 1006. The Corp - AccountBalance call can be used to map corporation wallet to appropriate accountKey.
//fromID	use 0 to skip, long	Optional upper bound for the transaction ID of returned transactions. This argument is normally used to walk to the transaction log backwards. See Journal Walking for more information.
//rowCount	int	Optional limit on number of rows to return. Default is 1000. Maximum is 2560.
func (api API) CorpWalletJournal(accountKey int64, fromID int64, rowCount int64) (*CorpWalletJournalResult, error) {
	output := CorpWalletJournalResult{}
	args := url.Values{}

	args.Add("accountKey", strconv.FormatInt(accountKey, 10))
	if fromID != 0 {
		args.Add("fromID", strconv.FormatInt(fromID, 10))
	}
	args.Add("rowCount", strconv.FormatInt(rowCount, 10))

	err := api.Call(CorpWalletJournalURL, args, &output)
	if err != nil {
		return nil, err
	}
	if output.Error != nil {
		return nil, output.Error
	}
	return &output, nil
}
