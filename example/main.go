package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/WulfgarW/sensonet"
	"github.com/ernesto-jimenez/httplogger"

	"github.com/eiannone/keyboard"
	"golang.org/x/oauth2"
)

const TOKEN_FILE = ".sensonet-token.json"
const CREDENTIALS_FILE = ".sensonet-credentials.json"

// Timeout is the default request timeout used by the Helper
var Timeout = 10 * time.Second

func readCredentials(filename string) (*sensonet.CredentialsStruct, error) {
	b, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var creds sensonet.CredentialsStruct
	err = json.Unmarshal(b, &creds)
	return &creds, err
}

func readToken(filename string) (*oauth2.Token, error) {
	b, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var token oauth2.Token
	err = json.Unmarshal(b, &token)
	return &token, err
}

func writeToken(filename string, token *oauth2.Token) error {
	b, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, b, 0o644)
}

type httpLogger struct {
	log *log.Logger
}

func newLogger(log *log.Logger) *httpLogger {
	return &httpLogger{
		log: log,
	}
}

func (l *httpLogger) LogRequest(req *http.Request) {
	l.log.Printf(
		"Request %s %s",
		req.Method,
		req.URL.String(),
	)
}

func (l *httpLogger) LogResponse(req *http.Request, res *http.Response, err error, duration time.Duration) {
	duration /= time.Millisecond
	if err != nil {
		l.log.Println(err)
	} else {
		l.log.Printf(
			"Response method=%s status=%d durationMs=%d %s",
			req.Method,
			res.StatusCode,
			duration,
			req.URL.String(),
		)
	}
}

// NewClient creates http client with default transport
func NewClient(log *log.Logger) *http.Client {
	return &http.Client{
		Timeout:   Timeout,
		Transport: httplogger.NewLoggedTransport(http.DefaultTransport, newLogger(log)),
	}
}

func readKey(input chan rune) {
	for {
		char, _, err := keyboard.GetSingleKey()
		//char, _, err := reader.ReadRune()
		if err != nil {
			log.Fatal(err)
		}
		input <- char
	}
}

func printKeyBinding() {
	fmt.Println("#############################################")
	fmt.Println("Choose an action:")
	fmt.Println("   1 = Read device and energy data")
	fmt.Println("   4 = Start hotwater boost")
	fmt.Println("   5 = Start zone quick veto")
	fmt.Println("   6 = Start strategy based quick mode")
	fmt.Println("   7 = Stop hotwater boost")
	fmt.Println("   8 = Stopt zone quick veto")
	fmt.Println("   9 = Stop strategy based quick mode")
	fmt.Println("   h = Show key bindings")
	fmt.Println("   q = Quit")
	fmt.Println("#############################################")
	fmt.Println("")
}

func main() {
	var (
		logger       = log.New(os.Stderr, "sensonet: ", log.Lshortfile)
		clientlogger = log.New(os.Stderr, "client: ", log.Lshortfile)
	)

	fmt.Println("Sample program to show how to use the sensonet library functions.")
	fmt.Println("")
	fmt.Println("")
	fmt.Println("First step: Reading credential file")
	// Read creadentials from file which are needed for the login
	credentials, err := readCredentials(CREDENTIALS_FILE)
	if err != nil {
		logger.Fatal(err)
	}

	fmt.Println("Second step: Reading token file")
	// Read token from file if a token is already present and stored in a file
	token, err := readToken(TOKEN_FILE)
	if err != nil {
		logger.Println("readToken() ended unsuccessful. Probably no token file found.")
	}

	fmt.Println("Third step: Generating new connection to be used for further calls of sensonet library")
	// Opens the connection to the myVaillant portal and returns a connection object for further function calls
	// If a token is provided, then it's validity is checked and a token refresh called if necessary.
	// If no token is provided or if a normal refresh is not possible, a login using the credentials is done
	clientlogger.SetOutput(io.Discard) //comment this out, if you want logging in http client
	client := NewClient(clientlogger)
	conn, newToken, err := sensonet.NewConnection(client, credentials, token)
	if err != nil {
		logger.Fatal(err)
	}

	// Store the current token in a file for future calls of this program
	if err := writeToken(TOKEN_FILE, newToken); err != nil {
		logger.Fatal(err)
	}

	fmt.Println("Fourth step: Reading Homes() structure from myVaillant portal")
	homes, err := conn.GetHomes()
	if err != nil {
		logger.Fatal(err)
	}
	//We use the system ID of the first element (=index 0) of homes[]
	systemID := homes[0].SystemID

	var heatingPar sensonet.HeatingParStruct
	var hotwaterPar sensonet.HotwaterParStruct
	heatingPar.ZoneIndex = 0
	heatingPar.VetoSetpoint = 18.0
	heatingPar.VetoDuration = -1.0 //negative value means: use default
	hotwaterPar.Index = -1         //negative value means: use default

	//Create a channel to read, if a key was pressed
	if err := keyboard.Open(); err != nil {
		panic(err)
	}
	input := make(chan rune, 1)
	go readKey(input)
	printKeyBinding()
	lastPrint := time.Now().Add(-25 * time.Second)

	for {
		select {
		case i := <-input:
			//fmt.Println(i)
			switch {
			case i == rune('1'):
				fmt.Println("Getting device data")
				devices, err := conn.GetDeviceData(systemID, sensonet.DEVICES_ALL)
				if err != nil {
					logger.Println(err)
				}
				fmt.Printf("   Got %d devices\n ", len(devices))
				fmt.Println("Reading energy data")
				startDate, _ := time.Parse("2006-01-02 15:04:05MST", "2024-11-01 00:00:00CET")
				endDate, _ := time.Parse("2006-01-02 15:04:05MST", "2024-11-20 23:59:59CET")
				for _, dev := range devices {
					for _, data := range dev.Device.Data {
						energyData, err := conn.GetEnergyData(systemID, dev.Device.DeviceUUID, data.OperationMode, data.ValueType, sensonet.RESOLUTION_DAY,
							startDate, endDate)
						if err != nil {
							logger.Println(err)
						} else {
							fmt.Printf("   Energy data for %s, %s, %s:\n", dev.Device.ProductName, data.OperationMode, data.ValueType)
							fmt.Printf("      %s bis %s: %.2f kWh\n", energyData.StartDate.Format("02.01.2006 15:04 MST"),
								energyData.EndDate.Format("02.01.2006 15:04 MST"), energyData.TotalConsumption/1000)
						}

					}
				}
			case i == rune('4'):
				fmt.Println("Starting hotwater boost")
				err = conn.StartHotWaterBoost(systemID, -1)
				if err != nil {
					logger.Println(err)
				}
			case i == rune('5'):
				fmt.Println("Starting zone quick veto")
				err = conn.StartZoneQuickVeto(systemID, 0, 18.0, 0.5)
				if err != nil {
					logger.Println(err)
				}
			case i == rune('6'):
				fmt.Println("Starting strategy based session")
				result, err := conn.StartStrategybased(systemID, sensonet.STRATEGY_HOTWATER_THEN_HEATING, &heatingPar, &hotwaterPar)
				if err != nil {
					logger.Println(err)
				} else {
					fmt.Printf("result=\"%s\"\n", result)
				}
			case i == rune('7'):
				fmt.Println("Stopping hotwater boost")
				err = conn.StopHotWaterBoost(systemID, -1)
				if err != nil {
					logger.Println(err)
				}
			case i == rune('8'):
				fmt.Println("Stopping zone quick veto")
				err = conn.StopZoneQuickVeto(systemID, 0)
				if err != nil {
					logger.Println(err)
				}
			case i == rune('9'):
				fmt.Println("Stopping strategy based session")
				result, err := conn.StopStrategybased(systemID, sensonet.STRATEGY_HOTWATER_THEN_HEATING, &heatingPar, &hotwaterPar)
				if err != nil {
					logger.Println(err)
				} else {
					fmt.Printf("result=\"%s\"\n", result)
				}
			case i == rune('h'):
				printKeyBinding()
			case i == rune('q'):
				_ = keyboard.Close()
				os.Exit(0)
			default:
				fmt.Println("You pressed a key without a function. Press h to get help")
			}
		default:
			// No key pressed. Print some information every 30 seconds
			if time.Now().After(lastPrint.Add(30 * time.Second)) {
				state, err := conn.GetSystem(systemID)
				if err != nil {
					logger.Fatal(err)
				}
				fmt.Println("---------------------------------------------------------------------------------------------------------------------")
				fmt.Printf("   OutdoorTemperature: %.1f°C\n", state.State.System.OutdoorTemperature)
				fmt.Print("   Zones: ")
				for i, c := range state.State.Zones {
					fmt.Printf("\"%s\":%.1f°C (Setpoint=%.1f°C), ", state.Configuration.Zones[i].General.Name, c.CurrentRoomTemperature, c.DesiredRoomTemperatureSetpoint)
				}
				fmt.Println("")
				if len(state.State.DomesticHotWater)*len(state.Configuration.DomesticHotWater) > 0 {
					fmt.Printf("   HotWaterTemperature: %.1f°C (Setpoint=%.1f°C)\n", state.State.DomesticHotWater[0].CurrentDomesticHotWaterTemperature, state.Configuration.DomesticHotWater[0].TappingSetpoint)
				}
				if len(state.State.Dhw)*len(state.Configuration.Dhw) > 0 {
					fmt.Printf("   HotWaterTemperature (Dhw): %.1f°C (Setpoint=%.1f°C)\n", state.State.Dhw[0].CurrentDhwTemperature, state.Configuration.Dhw[0].TappingSetpoint)
				}
				dhwData := sensonet.GetDhwData(state, -1)
				zoneData := sensonet.GetZoneData(state, heatingPar.ZoneIndex)
				fmt.Printf("   Quickmodes: internal: \"%s\"  heat pump: Dhw: \"%s\"  Zone: \"%s\"\n", conn.GetCurrentQuickMode(), dhwData.State.CurrentSpecialFunction, zoneData.State.CurrentSpecialFunction)
				fmt.Println("---------------------------------------------------------------------------------------------------------------------")
				lastPrint = time.Now()

			}
		}
	}

}
