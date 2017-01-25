package eveapi

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

const (
	//CharAccountBalanceURL is the url for the account balance endpoint
	CharAccountBalanceURL = "/char/AccountBalance.xml.aspx"
	//CharSkillQueueURL is the url for the skill queue endpoint
	CharSkillQueueURL = "/char/SkillQueue.xml.aspx"
	//MarketOrdersURL is the url for the market orders endpoint
	MarketOrdersURL = "/char/MarketOrders.xml.aspx"
	//WalletTransactionsURL is the url for the wallet transactions endpoint
	WalletTransactionsURL = "/char/WalletTransactions.xml.aspx"
	//CharacterSheetURL is the url for the character sheet endpoint
	CharacterSheetURL = "/char/CharacterSheet.xml.aspx"

	IndustryJobsURL = "/char/IndustryJobs.xml.aspx"

	ContractsURL = "/char/Contracts.xml.aspx"

	ContractItemsURL = "/char/ContractItems.xml.aspx"

	AssetListURL = "/char/AssetList.xml.aspx"
)

//AccountBalance is defined in corp.go

// CharAccountBalances calls /char/AccountBalance.xml.aspx
// Returns the account balance and any error if occured.
func (api API) CharAccountBalances(charID int64) (*AccountBalance, error) {
	output := AccountBalance{}
	arguments := url.Values{}
	arguments.Add("characterID", strconv.FormatInt(charID, 10))
	err := api.Call(CharAccountBalanceURL, arguments, &output)
	if err != nil {
		return nil, err
	}
	if output.Error != nil {
		return nil, output.Error
	}
	return &output, nil
}

//SkillQueueRow is an entry in a character's skill queue
type SkillQueueRow struct {
	Position  int     `xml:"queuePosition,attr"`
	TypeID    int     `xml:"typeID,attr"`
	Level     int     `xml:"level,attr"`
	StartSP   int     `xml:"startSP,attr"`
	EndSP     int     `xml:"endSP,attr"`
	StartTime eveTime `xml:"startTime,attr"`
	EndTime   eveTime `xml:"endTime,attr"`
}

func (s SkillQueueRow) String() string {
	return fmt.Sprintf("Position: %v, TypeID: %v, Level: %v, StartSP: %v, EndSP: %v, StartTime: %v, EndTime: %v", s.Position, s.TypeID, s.Level, s.StartSP, s.EndSP, s.StartTime, s.EndTime)
}

//SkillQueueResult is the result returned by the skill queue endpoint
type SkillQueueResult struct {
	APIResult
	SkillQueue []SkillQueueRow `xml:"result>rowset>row"`
}

// SkillQueue calls the API passing the parameter charID
// Returns a SkillQueueResult struct
func (api API) SkillQueue(charID int64) (*SkillQueueResult, error) {
	output := SkillQueueResult{}
	arguments := url.Values{}
	arguments.Add("characterID", strconv.FormatInt(charID, 10))
	err := api.Call(CharSkillQueueURL, arguments, &output)
	if err != nil {
		return nil, err
	}
	if output.Error != nil {
		return nil, output.Error
	}
	return &output, nil
}

//MarketOrdersResult is the result from calling the market orders endpoint
type MarketOrdersResult struct {
	APIResult
	Orders []MarketOrder `xml:"result>rowset>row"`
}

//MarketOrder is either a sell order or buy order
type MarketOrder struct {
	OrderID      int     `xml:"orderID,attr"`
	CharID       int64   `xml:"charID,attr"`
	StationID    int64   `xml:"stationID,attr"`
	VolEntered   int     `xml:"volEntered,attr"`
	VolRemaining int64   `xml:"volRemaining,attr"`
	MinVolume    int     `xml:"minVolume,attr"`
	TypeID       int64   `xml:"typeID,attr"`
	Range        int     `xml:"range,attr"`
	Division     int     `xml:"accountKey,attr"`
	Escrow       float64 `xml:"escrow,attr"`
	Price        float64 `xml:"price,attr"`
	IsBuyOrder   bool    `xml:"bid,attr"`
	Issued       eveTime `xml:"issued,attr"`
	Duration     int     `xml:"duration,attr"`
    OrderState   int64   `xml:"orderState,attr"`

}

//MarketOrders returns the market orders for a given character
func (api API) MarketOrders(charID int64) (*MarketOrdersResult, error) {
	output := MarketOrdersResult{}
	args := url.Values{}
	args.Add("characterID", strconv.FormatInt(charID, 10))
	err := api.Call(MarketOrdersURL, args, &output)
	if err != nil {
		return nil, err
	}
	if output.Error != nil {
		return nil, output.Error
	}
	return &output, nil
}

//MarketOrder is either a sell order or buy order
type WalletTransaction struct {
	TransactionDateTime  eveTime `xml:"transactionDateTime,attr"`  //datetime  Date and time of transaction.
	TransactionID        int64   `xml:"transactionID,attr"`        //long      Unique transaction ID.
	Quantity             int64   `xml:"quantity,attr"`             //int       Number of items bought or sold.
	TypeName             string  `xml:"typeName,attr"`             //string    Name of item bought or sold.
	TypeID               int64   `xml:"typeID,attr"`               //int       Type ID of item bought or sold.
	Price                float64 `xml:"price,attr"`                //decimal   Amount paid per unit.
	ClientID             int64   `xml:"clientID,attr"`             //long      Counterparty character or corporation ID. For NPC corporations, see the appropriate cross reference.
	ClientName           string  `xml:"clientName,attr"`           //string    Counterparty name.
	StationID            int64   `xml:"stationID,attr"`            //long      Station ID in which transaction took place.
	StationName          string  `xml:"stationName,attr"`          //string    Name of station in which transaction took place.
	TransactionType      string  `xml:"transactionType,attr"`      //string    Either "buy" or "sell" as appropriate.
	TransactionFor       string  `xml:"transactionFor,attr"`       //string    Either "personal" or "corporate" as appropriate.
	JournalTransactionID int64   `xml:"journalTransactionID,attr"` //long      Corresponding wallet journal refID.
	ClientTypeID         int64   `xml:"clientTypeID,attr"`         //long      Unknown meaning/mapping.
}
type WalletTransactionsResult struct {
	APIResult
	Transactions []WalletTransaction `xml:"result>rowset>row"`
}

//WalletTransactions returns the wallet transactions for a given character
//characterID	long	Character ID for which transactions will be requested
//accountKey	int	Account key of the wallet for which transactions will be returned. This is optional for character accounts which only have one wallet (accountKey = 1000). However, corporations have seven wallets with accountKeys numbered from 1000 through 1006. The Corp - AccountBalance call can be used to map corporation wallet to appropriate accountKey.
//fromID	use 0 to skip, long	Optional upper bound for the transaction ID of returned transactions. This argument is normally used to walk to the transaction log backwards. See Journal Walking for more information.
//rowCount	int	Optional limit on number of rows to return. Default is 1000. Maximum is 2560.
func (api API) WalletTransactions(charID int64, accountKey int64, fromID int64, rowCount int64) (*WalletTransactionsResult, error) {
	output := WalletTransactionsResult{}
	args := url.Values{}

	args.Add("characterID", strconv.FormatInt(charID, 10))
	args.Add("accountKey", strconv.FormatInt(accountKey, 10))
	if fromID != 0 {
		args.Add("fromID", strconv.FormatInt(fromID, 10))
	}
	args.Add("rowCount", strconv.FormatInt(rowCount, 10))

	err := api.Call(WalletTransactionsURL, args, &output)
	if err != nil {
		return nil, err
	}
	if output.Error != nil {
		return nil, output.Error
	}
	return &output, nil
}

func (api API) SimpleWalletTransactions(charID int64, fromID int64) (*WalletTransactionsResult, error) {
	return api.WalletTransactions(charID, 1000, fromID, 2560)
}

type Row struct {
	TypeID      int64 `xml:"typeID,attr"`
	Published   bool  `xml:"published,attr"`
	Level       int64 `xml:"level,attr"`
	SkillPoints int64 `xml:"skillpoints,attr"`
}

type Rowset struct {
	Name string `xml:"name,attr"`
	Rows []Row  `xml:"row"`
}

type CharacterSheetResult struct {
	APIResult
	Rowsets []Rowset `xml:"result>rowset"`
	Skills  []Row
}

func (api API) CharacterSheet(charID int64) (*CharacterSheetResult, error) {
	output := CharacterSheetResult{}
	args := url.Values{}
	args.Add("characterID", strconv.FormatInt(charID, 10))
	err := api.Call(CharacterSheetURL, args, &output)
	if err != nil {
		return nil, err
	}
	if output.Error != nil {
		return nil, output.Error
	}
	for _, v := range output.Rowsets {
		if v.Name == "skills" {
			output.Skills = v.Rows
		}
	}
	return &output, nil
}

type Job struct {
	JobID                int64   `xml:"jobID,attr"`
	InstallerID          int64   `xml:"installerID,attr"`
	InstallerName        string  `xml:"installerName,attr"`
	FacilityID           int64   `xml:"facilityID,attr"`
	SolarSystemID        int64   `xml:"solarSystemID,attr"`
	SolarSystemName      string  `xml:"solarSystemName,attr"`
	StationID            int64   `xml:"stationID,attr"`
	ActivityID           int64   `xml:"activityID,attr"` //1 - mnf 4 - im.me
	BlueprintID          int64   `xml:"blueprintID,attr"`
	BlueprintTypeID      int64   `xml:"blueprintTypeID,attr"`
	BlueprintTypeName    string  `xml:"blueprintTypeName,attr"`
	BlueprintLocationID  int64   `xml:"blueprintLocationID,attr"`
	OutputLocationID     int64   `xml:"outputLocationID,attr"`
	Runs                 int64   `xml:"runs,attr"`
	Cost                 float64 `xml:"cost,attr"`
	TeamID               int64   `xml:"teamID,attr"`
	LicensedRuns         int64   `xml:"licensedRuns,attr"`
	Probability          int64   `xml:"probability,attr"`
	ProductTypeID        int64   `xml:"productTypeID,attr"`
	ProductTypeName      string  `xml:"productTypeName,attr"`
	Status               int64   `xml:"status,attr"`
	TimeInSeconds        int64   `xml:"timeInSeconds,attr"`
	StartDate            eveTime `xml:"startDate,attr"`
	EndDate              eveTime `xml:"endDate,attr"`
	PauseDate            eveTime `xml:"pauseDate,attr"`
	CompletedDate        eveTime `xml:"completedDate,attr"`
	CompletedCharacterID int64   `xml:"completedCharacterID,attr"`
	SuccessfulRuns       int64   `xml:"successfulRuns,attr"`
}

type IndustryJobsResult struct {
	APIResult
	Jobs []Job `xml:"result>rowset>row"`
}

func (api API) IndustryJobs(charID int64) (*IndustryJobsResult, error) {
	output := IndustryJobsResult{}
	args := url.Values{}
	args.Add("characterID", strconv.FormatInt(charID, 10))
	err := api.Call(IndustryJobsURL, args, &output)
	if err != nil {
		return nil, err
	}
	if output.Error != nil {
		return nil, output.Error
	}
	return &output, nil
}

type Contract struct {
	ContractID     int64   `xml:"contractID,attr"`     //Unique Identifier for the contract.
	IssuerID       int64   `xml:"issuerID,attr"`       //Character ID for the issuer.
	IssuerCorpID   int64   `xml:"issuerCorpID,attr"`   //Characters corporation ID for the issuer.
	AssigneeID     int64   `xml:"assigneeID,attr"`     //ID to whom the contract is assigned, can be corporation or character ID.
	AcceptorID     int64   `xml:"acceptorID,attr"`     //Who will accept the contract. If assigneeID is same as acceptorID then CharacterID else CorporationID (The contract accepted by the corporation).
	StartStationID int32   `xml:"startStationID,attr"` //Start station ID (for Couriers contract).
	EndStationID   int32   `xml:"endStationID,attr"`   //End station ID (for Couriers contract).
	Type           string  `xml:"type,attr"`           //Type of the contract (ItemExchange, Courier, Loan or Auction).
	Status         string  `xml:"status,attr"`         //Status of the the contract (Outstanding, Deleted, Completed, Failed, CompletedByIssuer, CompletedByContractor, Cancelled, Rejected, Reversed or InProgress)
	Title          string  `xml:"title,attr"`          //Title of the contract
	ForCorp        int32   `xml:"forCorp,attr"`        //1 if the contract was issued on behalf of the issuer's corporation, 0 otherwise
	Availability   string  `xml:"availability,attr"`   //Public or Private
	DateIssued     eveTime `xml:"dateIssued,attr"`     //Сreation date of the contract
	DateExpired    eveTime `xml:"dateExpired,attr"`    //Expiration date of the contract
	DateAccepted   eveTime `xml:"dateAccepted,attr"`   //Date of confirmation of contract
	NumDays        int32   `xml:"numDays,attr"`        //Number of days to perform the contract
	DateCompleted  eveTime `xml:"dateCompleted,attr"`  //Date of completed of contract
	Price          float64 `xml:"price,attr"`          //Price of contract (for ItemsExchange and Auctions)
	Reward         float64 `xml:"reward,attr"`         //Remuneration for contract (for Couriers only)
	Collateral     float64 `xml:"collateral,attr"`     //Collateral price (for Couriers only)
	Buyout         float64 `xml:"buyout,attr"`         //Buyout price (for Auctions only)
	Volume         float64 `xml:"volume,attr"`         //Volume of items in the contract

	IssuerName     string
	IssuerCorpName string
	ContractItems  []ContractItem
}

type ContractsResult struct {
	APIResult
	Contracts []Contract `xml:"result>rowset>row"`
}

func (api API) Contracts(charID int64) (*ContractsResult, error) {
	output := ContractsResult{}
	args := url.Values{}
	args.Add("characterID", strconv.FormatInt(charID, 10))
	err := api.Call(ContractsURL, args, &output)
	if err != nil {
		return nil, err
	}
	if output.Error != nil {
		return nil, output.Error
	}

	ids := make(map[int64]int64)
	jIds := make([]string, 0)
	for _, c := range output.Contracts {
		_, exists := ids[c.IssuerID]
		if !exists {
			ids[c.IssuerID] = 1
			jIds = append(jIds, strconv.FormatInt(c.IssuerID, 10))
		}
		_, exists = ids[c.IssuerCorpID]
		if !exists {
			ids[c.IssuerCorpID] = 1
			jIds = append(jIds, strconv.FormatInt(c.IssuerCorpID, 10))
		}
	}

	names, _ := api.IdsToNames(strings.Join(jIds, ","))

	for i, ct := range output.Contracts {
		for _, rec := range names.Names {
			if ct.IssuerID == rec.ID {
				output.Contracts[i].IssuerName = rec.Name
			}
			if ct.IssuerCorpID == rec.ID {
				output.Contracts[i].IssuerCorpName = rec.Name
			}
		}

		items, _ := api.ContractItems(charID, ct.ContractID)
		output.Contracts[i].ContractItems = items.ContractItems
	}

	return &output, nil
}

type ContractItem struct {
	RecordID    int64 `xml:"recordID,attr"`    // Unique Identifier for the contract.
	TypeID      int64 `xml:"typeID,attr"`      // Type ID for item.
	Quantity    int64 `xml:"quantity,attr"`    // Number of items in the stack.
	RawQuantity int64 `xml:"rawQuantity,attr"` // This attribute will only show up if the quantity is a negative number in the DB. Negative quantities are in fact codes, -1 indicates that the item is a singleton (non-stackable). If the item happens to be a Blueprint, -1 is an Original and -2 is a Blueprint Copy.
	Singleton   int64 `xml:"singleton,attr"`   // 1 if this is a singleton item, 0 if not.
	Included    int64 `xml:"included,attr"`    // 1 if the contract issuer has submitted this item with the contract, 0 if the isser is asking for this item in the contract.
}
type ContractItemsResult struct {
	APIResult
	ContractItems []ContractItem `xml:"result>rowset>row"`
}

func (api API) ContractItems(charID int64, contractID int64) (*ContractItemsResult, error) {
	output := ContractItemsResult{}
	args := url.Values{}
	args.Add("characterID", strconv.FormatInt(charID, 10))
	args.Add("contractID", strconv.FormatInt(contractID, 10))
	err := api.Call(ContractItemsURL, args, &output)
	if err != nil {
		return nil, err
	}
	if output.Error != nil {
		return nil, output.Error
	}
	return &output, nil
}

type Asset struct {
	ItemID      int64 `xml:"itemID,attr"`      // Unique ID for this item. The ID of an item is stable if that item is not repackaged, stacked, detached from a stack, assembled, or otherwise altered. If an item is changed in one of these ways, then the ID will also change (see notes below).
	LocationID  int64 `xml:"locationID,attr"`  // References a solar system or station. Note that in the nested XML response this column is not present in the sub-asset lists, as those share the locationID of their parent node. Example: a module in a container in a ship in a station.. Whereas the flat XML returns a locationID for each item. (See the notes on how to resolve the locationID to a solar system or station)
	TypeID      int64 `xml:"typeID,attr"`      // The type of this item.
	Quantity    int64 `xml:"quantity,attr"`    // How many items are in this stack.
	Flag        int64 `xml:"flag,attr"`        // Indicates something about this item's storage location. The flag is used to differentiate between hangar divisions, drone bay, fitting location, and similar. Please see the Inventory Flags documentation.
	Singleton   bool  `xml:"singleton,attr"`   // If True (1), indicates that this item is a singleton. This means that the item is not packaged.
	RawQuantity int64 `xml:"rawQuantity,attr"` // Items in the AssetList (and ContractItems) now include a rawQuantity attribute if the quantity in the DB is negative (see notes).

	Assets []Asset `xml:"rowset>row"`
}
type AssetListResult struct {
	APIResult
	Assets []Asset `xml:"result>rowset>row"`
}

func (api API) AssetList(charID int64, flat int64) (*AssetListResult, error) {
	output := AssetListResult{}
	args := url.Values{}
	args.Add("characterID", strconv.FormatInt(charID, 10))
	args.Add("flat", strconv.FormatInt(flat, 10))
	err := api.Call(AssetListURL, args, &output)
	if err != nil {
		return nil, err
	}
	if output.Error != nil {
		return nil, output.Error
	}
	return &output, nil
}
