package instruments

import (
	"encoding/json"
	"fmt"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
	"log"
	"strconv"
	"strings"
)

type MusicalInstrumentSales struct {
	methods map[string]func(APIstub shim.ChaincodeStubInterface, args []string) pb.Response
}

type Instrument struct {
	Type     string  `json:"type"`
	Brand    string  `json:"brand"`
	Model    string  `json:"model"`
	Color    string  `json:"color"`
	YearMade int     `json:"yearMade"`
	Owner    string  `json:"owner"`
	Price    float64 `json:"price"`
	SerialNo string  `json:"serialNo"`
}

func main() {
	err := shim.Start(new(MusicalInstrumentSales))
	if err != nil {
		fmt.Printf("Error starting MusicalInstrumentSales chaincode: %s", err)
	}
}

func (m *MusicalInstrumentSales) Init(stub shim.ChaincodeStubInterface) pb.Response {
	m.methods = map[string]func(APIstub shim.ChaincodeStubInterface, args []string) pb.Response{
		"initInstrument":     m.initInstrument,
		"transferInstrument": m.transferInstrument,
		"readInstrument":     m.readInstrument,
		"deleteInstrument":   m.deleteInstrument,
	}
	return shim.Success(nil)
}

func (m *MusicalInstrumentSales) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	function, args := stub.GetFunctionAndParameters()

	method := m.methods[function]
	if method != nil {
		log.Println("invoked method not found: " + function)
		return m.initInstrument(stub, args)
	}
	return shim.Error("Unknown function invocation")
}

func (m *MusicalInstrumentSales) initInstrument(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var err error
	if len(args) != 8 {
		return shim.Error("Incorrect number of arguments. Expecting 8")
	}
	for i := range args {
		if args[i] == "" {
			return shim.Error("argument no#" + strconv.Itoa(i+1) + " must be a non-empty string")
		}
	}
	typ := args[0]
	brand := args[1]
	model := args[2]
	color := strings.ToLower(args[3])
	year, err := strconv.Atoi(args[4])
	if err != nil {
		return shim.Error("argument 5 must be a numeric string")
	}
	owner := args[5]
	price, err := strconv.ParseFloat(args[6], 64)
	if err != nil {
		return shim.Error("argument 7 must be a numeric/double string")
	}
	serialNo := args[7]

	instrumentAsBytes, err := stub.GetState(serialNo)
	if err != nil {
		return shim.Error("Failed to get instrument: " + err.Error())
	} else if instrumentAsBytes != nil {
		fmt.Println("This instrument already exists: " + model)
		return shim.Error("This instrument already exists: " + model)
	}
	instrument := &Instrument{
		typ,
		brand,
		model,
		color,
		year,
		owner,
		price,
		serialNo,
	}
	instrumentJSONasBytes, err := json.Marshal(instrument)
	if err != nil {
		return shim.Error(err.Error())
	}
	err = stub.PutState(serialNo, instrumentJSONasBytes)
	if err != nil {
		return shim.Error(err.Error())
	}

	indexName := "brand~serialNo"
	colorNameIndexKey, err := stub.CreateCompositeKey(indexName, []string{instrument.Brand, instrument.SerialNo})
	if err != nil {
		return shim.Error(err.Error())
	}
	value := []byte{0x00}
	stub.PutState(colorNameIndexKey, value)
	return shim.Success(nil)
}

func (m *MusicalInstrumentSales) readInstrument(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var serialNo, jsonResp string
	var err error

	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting serial number of the instrument to query")
	}
	serialNo = args[0]
	valAsbytes, err := stub.GetState(serialNo)
	if err != nil {
		jsonResp = "{\"Error\":\"Failed to get state for " + serialNo + "\"}"
		return shim.Error(jsonResp)
	} else if valAsbytes == nil {
		jsonResp = "{\"Error\":\"Instrument does not exist: " + serialNo + "\"}"
		return shim.Error(jsonResp)
	}
	return shim.Success(valAsbytes)
}

func (m *MusicalInstrumentSales) deleteInstrument(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var jsonResp string
	var instrumentJSON Instrument
	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}
	serialNo := args[0]

	valAsbytes, err := stub.GetState(serialNo)
	if err != nil {
		jsonResp = "{\"Error\":\"Failed to get state for " + serialNo + "\"}"
		return shim.Error(jsonResp)
	} else if valAsbytes == nil {
		jsonResp = "{\"Error\":\"Instrument does not exist: " + serialNo + "\"}"
		return shim.Error(jsonResp)
	}

	err = json.Unmarshal([]byte(valAsbytes), &instrumentJSON)
	if err != nil {
		jsonResp = "{\"Error\":\"Failed to decode JSON of: " + serialNo + "\"}"
		return shim.Error(jsonResp)
	}

	err = stub.DelState(serialNo)
	if err != nil {
		return shim.Error("Failed to delete state:" + err.Error())
	}

	indexName := "brand~serialNo"
	brandSerialNoIndexKey, err := stub.CreateCompositeKey(indexName, []string{instrumentJSON.Brand, instrumentJSON.SerialNo})
	if err != nil {
		return shim.Error(err.Error())
	}

	err = stub.DelState(brandSerialNoIndexKey)
	if err != nil {
		return shim.Error("Failed to delete state:" + err.Error())
	}
	return shim.Success(nil)
}

func (m *MusicalInstrumentSales) transferInstrument(stub shim.ChaincodeStubInterface, args []string) pb.Response {

	if len(args) < 2 {
		return shim.Error("Incorrect number of arguments. Expecting 2")
	}

	serialNo := args[0]
	newOwner := strings.ToLower(args[1])

	instrumentAsBytes, err := stub.GetState(serialNo)
	if err != nil {
		return shim.Error("Failed to get instrument:" + err.Error())
	} else if instrumentAsBytes == nil {
		return shim.Error("Instrument does not exist")
	}

	instrumentToTransfer := Instrument{}
	err = json.Unmarshal(instrumentAsBytes, &instrumentToTransfer)
	if err != nil {
		return shim.Error(err.Error())
	}
	instrumentToTransfer.Owner = newOwner

	instrumentJSONasBytes, _ := json.Marshal(instrumentToTransfer)
	err = stub.PutState(serialNo, instrumentJSONasBytes)
	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success(nil)
}
