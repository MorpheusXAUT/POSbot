package util

type StarbaseFuel struct {
	TypeID int
	Amount int
}

var (
	StarbaseFuelRequired map[int]StarbaseFuel = map[int]StarbaseFuel{
		// Amarr
		12235: {
			TypeID: 4247,
			Amount: 40,
		},
		20059: {
			TypeID: 4247,
			Amount: 20,
		},
		20060: {
			TypeID: 4247,
			Amount: 10,
		},
		// Caldari
		16213: {
			TypeID: 4051,
			Amount: 40,
		},
		20061: {
			TypeID: 4051,
			Amount: 20,
		},
		20062: {
			TypeID: 4051,
			Amount: 10,
		},
		// Gallente
		12236: {
			TypeID: 4312,
			Amount: 40,
		},
		20063: {
			TypeID: 4312,
			Amount: 20,
		},
		20064: {
			TypeID: 4312,
			Amount: 10,
		},
		// Minmatar
		16214: {
			TypeID: 4316,
			Amount: 40,
		},
		20065: {
			TypeID: 4316,
			Amount: 20,
		},
		20066: {
			TypeID: 4316,
			Amount: 10,
		},
	}
)

func FuelRequiredForStarbase(typeID int) StarbaseFuel {
	fuel, ok := StarbaseFuelRequired[typeID]
	if !ok {
		return StarbaseFuel{
			TypeID: 0,
			Amount: 0,
		}
	}
	return fuel
}
