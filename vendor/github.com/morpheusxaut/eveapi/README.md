#eveapi
[![GoDoc](https://godoc.org/github.com/flexd/eveapi?status.svg)](https://godoc.org/github.com/flexd/eveapi)
## EVE Online API Client
eveapi is a library that provides access to the EVE Online XML API.

This needs more work, ALPHA STATUS.
Barely anything is done, API subject to change.

## Todo


* Caching - github.com/inominate/eve-api-proxy
* Everything else
* More things

## Usage

Below is an example which shows some of the calls available currently.
```go
package main

import (
    "fmt"
    "log"

    "github.com/flexd/eveapi"
)

func main() {
    key := eveapi.Key{"somekey", "somevcode"}
    charID := "93014296"
    voidCharID := "93947594"
    api := &eveapi.API{
    //Server:    eveapi.Tranquility,
    //github.com/inominate/eve-api-proxy
    Server:    eveapi.LocalProxy,
    APIKey:    key,
    UserAgent: "Hello",
    Debug:     false,
    }
    //api := eveapi.Simple(key)
	serverStatus, err := api.ServerStatus()
	if err != nil {
		log.Fatalln(err)
		return
	}
	fmt.Println("Online:", serverStatus.Open, "Players:", serverStatus.OnlinePlayers)
	charnames, err := api.AccountGetChars()
	if err != nil {
		log.Fatalln(err)
		return
	}

	characters, err := api.Names2ID(strings.Join(charnames[:], ","))
	if err != nil {
		log.Fatalln(err)
		return
	}
	fmt.Println("Characters:", characters)
	fmt.Println("First char:", characters[0])
	accounts, err := api.CharAccountBalances(characters[0].ID)
	if err != nil {
		log.Fatalln(err)
		return
	}
	fmt.Println("Current time:", accounts.CurrentTime, "Cached until:", accounts.CachedUntil)
	for _, c := range accounts.Accounts {
		fmt.Println("AccountID:", c.ID, "Key:", c.Key, "Balance:", c.Balance, "ISK")
	}
	accountStatus, err := api.AccountStatus()
	if err != nil {
		log.Fatalln(err)
		return
	}
	fmt.Println("Account is paid until: ", accountStatus.PaidUntil)
	fmt.Println("Account was created: ", accountStatus.CreateDate)
	fmt.Println("Number of times Logged in: ", accountStatus.LogonCount, " Length of Time Logged in: ", accountStatus.LogonMinutes)
	skillqueue, err := api.SkillQueue(characters[0].ID)
	if err != nil {
		log.Fatalln(err)
		return
	}
	for _, sq := range skillqueue.SkillQueue {
		fmt.Println(sq)
	}
}
```
