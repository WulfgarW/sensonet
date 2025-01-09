package main

// This is an example program to demonstrate the usage of the package.
// As it is recommended to usage the controller (initiated by NewController()) and its methods, this example program does not demonstrate the methods
// of the connection object.

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/WulfgarW/sensonet"
	"github.com/ernesto-jimenez/httplogger"

	"github.com/eiannone/keyboard"
	"golang.org/x/oauth2"
)

const LOG_FILE = "sensonet.log"
const TOKEN_FILE = ".sensonet-token.json"
const CREDENTIALS_FILE = ".sensonet-credentials.json"
const WITH_SENSONET_LOGGING = true    // Set this to false if you want no sensonet logging
const WITH_HTTP_CLIENT_LOGGING = true // Set this to false if you want no http client logging in the sensonet library

// Timeout is the default request timeout used by the http client
var Timeout = 45 * time.Second // Fetching energy data can take some time

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

func readKey(input chan rune) {
	for {
		char, _, err := keyboard.GetSingleKey()
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
	fmt.Println("   8 = Stop zone quick veto")
	fmt.Println("   9 = Stop strategy based quick mode")
	fmt.Println("   0 = Read mpc data")
	fmt.Println("   h = Show key bindings")
	fmt.Println("   q = Quit")
	fmt.Println("#############################################")
	fmt.Println("")
}

// Implementation of log functions for the http client in the sensonet library
// (not necessary, if you don't want to log the http client in the sensonet library)
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

func NewClientWithLog(log *log.Logger) *http.Client {
	return &http.Client{
		Timeout:   Timeout,
		Transport: httplogger.NewLoggedTransport(http.DefaultTransport, newLogger(log)),
	}
}

func NewClient() *http.Client {
	return &http.Client{
		Timeout: Timeout,
	}
}

// Implementation of log functions for the logger interface of the sensonet library
// (not necessary, if you don't want to use the logger interface)
type SLogger struct {
	logger *log.Logger
}

func NewSLogLogger(logFile *os.File) *SLogger {
	logger := log.New(logFile, "sensonetlogger: ", log.Lshortfile)
	return &SLogger{logger: logger}
}

func (l *SLogger) Printf(msg string, arg ...any) {
	l.logger.Println(fmt.Sprint("Debug: ", msg, arg))
}

// Main program
func main() {
	var logFile *os.File
	var err error
	if LOG_FILE != "" {
		logFile, err = os.OpenFile(LOG_FILE, os.O_CREATE|os.O_WRONLY, 0666)
		if err != nil {
			fmt.Println("Error opening log file. Err %w", err)
			os.Exit(1)
		}
	} else {
		logFile = os.Stderr
	}

	var (
		logger = log.New(logFile, "sensonet: ", log.Lshortfile)
	)

	fmt.Println("Sample program to show how to use the sensonet library functions.")
	fmt.Println("")
	fmt.Println("")
	fmt.Println("First step: Reading credential file")
	// Read creadentials from file which are needed for the login
	credentials, err := readCredentials(CREDENTIALS_FILE)
	if err != nil {
		logger.Println("readCredentials() ended unsuccessful. Probably no credential file was found. Error:", err)
	} else {
		fmt.Println("Read credentials from file")
	}

	fmt.Println("Second step: Reading token file")
	// Read token from file if a token is already present and stored in a file
	token, err := readToken(TOKEN_FILE)
	if err != nil {
		logger.Println("readToken() ended unsuccessful. Probably no token file was found. Error:", err)
	} else {
		logger.Println("Token read from file:", token)
	}

	fmt.Println("Third step: Generating new connection to be used for further calls of sensonet library")

	var client *http.Client
	client = NewClient()

	// If you have user, password and realm, use Oauth2ConfigForRealm() and PasswordCredentialsToken() to get a token
	ctx := context.WithValue(context.TODO(), oauth2.HTTPClient, client)
	clientCtx := context.WithValue(ctx, oauth2.HTTPClient, client)
	oc := sensonet.Oauth2ConfigForRealm(credentials.Realm)
	token, err = oc.PasswordCredentialsToken(clientCtx, credentials.User, credentials.Password)
	if err != nil {
		logger.Fatal(err)
	}

	// If http client logging is wanted, you have to prepare an http client with logging
	if WITH_HTTP_CLIENT_LOGGING {
		clientlogger := log.New(logFile, "client: ", log.Lshortfile)
		client = NewClientWithLog(clientlogger)
	}

	// NewConnection() opens the connection to the myVaillant portal and returns a connection object for further function calls.
	// You can provide an http client (especially one with logging) as optional parameter.
	var conn *sensonet.Connection
	if WITH_HTTP_CLIENT_LOGGING {
		conn, err = sensonet.NewConnection(oc.TokenSource(clientCtx, token), sensonet.WithHttpClient(client))
	} else {
		conn, err = sensonet.NewConnection(oc.TokenSource(clientCtx, token))
	}
	if err != nil {
		logger.Fatal(err)
	}

	// NewController() initialises a controller that caches data read from the myVaillant portal and provides functions to control the heat pump system.
	// You can provide a logger as optional parameter.
	var ctrl *sensonet.Controller
	if WITH_SENSONET_LOGGING {
		// Implements a logger for the sensonet library
		slogger := NewSLogLogger(logFile)
		ctrl, err = sensonet.NewController(conn, sensonet.WithLogger(slogger))
	} else {
		ctrl, err = sensonet.NewController(conn)
	}
	if err != nil {
		logger.Fatal(err)
	}

	// Store the current token in a file for future calls of this program
	if err := writeToken(TOKEN_FILE, token); err != nil {
		logger.Fatal(err)
	} else {
		logger.Println("New Token written to file:", token)
	}

	fmt.Println("Fourth step: Reading Homes() structure from myVaillant portal")
	homes, err := ctrl.GetHomes()
	if err != nil {
		logger.Fatal(err)
	}
	// We use the system ID of the first element (=index 0) of homes[]
	systemId := homes[0].SystemID

	var heatingPar sensonet.HeatingParStruct
	var hotwaterPar sensonet.HotwaterParStruct
	heatingPar.ZoneIndex = 0
	heatingPar.VetoSetpoint = 18.0
	heatingPar.VetoDuration = -1.0 // negative value means: use default
	hotwaterPar.Index = -1         // negative value means: use default

	// Create a channel to read, if a key was pressed
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
			switch {
			case i == rune('1'):
				fmt.Println("Getting device data")
				devices, err := ctrl.GetDeviceData(systemId, sensonet.DEVICES_ALL)
				if err != nil {
					logger.Println(err)
				}
				fmt.Printf("   Got %d devices\n ", len(devices))
				fmt.Println("Reading energy data")
				startDate, _ := time.Parse("2006-01-02 15:04:05MST", "2025-01-01 00:00:00CET")
				endDate, _ := time.Parse("2006-01-02 15:04:05MST", "2025-01-10 23:59:59CET")
				for _, dev := range devices {
					for _, data := range dev.Device.Data {
						energyData, err := ctrl.GetEnergyData(systemId, dev.Device.DeviceUUID, data.OperationMode, data.ValueType, sensonet.RESOLUTION_DAY,
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
				err = ctrl.StartHotWaterBoost(systemId, -1)
				if err != nil {
					fmt.Println(" An error occurred. ", err)
					logger.Println(err)
				}
			case i == rune('5'):
				fmt.Println("Starting zone quick veto")
				err = ctrl.StartZoneQuickVeto(systemId, 0, 18.0, 0.5)
				if err != nil {
					fmt.Println(" An error occurred. ", err)
					logger.Println(err)
				}
			case i == rune('6'):
				fmt.Println("Starting strategy based session")
				result, err := ctrl.StartStrategybased(systemId, sensonet.STRATEGY_HOTWATER_THEN_HEATING, &heatingPar, &hotwaterPar)
				if err != nil {
					fmt.Println(" An error occurred. ", err)
					logger.Println(err)
				} else {
					fmt.Printf("result of function StartStrategybased()=\"%s\"\n", result)
				}
			case i == rune('7'):
				fmt.Println("Stopping hotwater boost")
				err = ctrl.StopHotWaterBoost(systemId, -1)
				if err != nil {
					fmt.Println(" An error occurred. ", err)
					logger.Println(err)
				}
			case i == rune('8'):
				fmt.Println("Stopping zone quick veto")
				err = ctrl.StopZoneQuickVeto(systemId, 0)
				if err != nil {
					fmt.Println(" An error occurred. ", err)
					logger.Println(err)
				}
			case i == rune('9'):
				fmt.Println("Stopping strategy based session")
				result, err := ctrl.StopStrategybased(systemId, &heatingPar, &hotwaterPar)
				if err != nil {
					fmt.Println(" An error occurred. ", err)
					logger.Println(err)
				} else {
					fmt.Printf("result of function StopStrategybased()=\"%s\"\n", result)
				}
			case i == rune('0'):
				fmt.Println("Getting mpc data")
				result, err := ctrl.GetMpcData(systemId)
				if err != nil {
					fmt.Println(" An error occurred. ", err)
					logger.Println(err)
				} else {
					logger.Println("result of function GetMpcData()=", result)
				}
				systemCurrentPower, err := ctrl.SystemCurrentPower(systemId)
				if err != nil {
					fmt.Println(" An error occurred. ", err)
					logger.Println(err)
				} else {
					fmt.Println("Current power consumption of system=", systemCurrentPower)
				}
				devicePowerMap, err := ctrl.DeviceCurrentPower(systemId, "All")
				if err != nil {
					fmt.Println(" An error occurred. ", err)
					logger.Println(err)
				} else {
					fmt.Println("Current power consumption of all devices of the system:")
					for _, el := range devicePowerMap {
						fmt.Printf("   Device: %-35s Current power: %5.0fW \n", el.ProductName, el.CurrentPower)
					}
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
				state, err := ctrl.GetSystem(systemId)
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
				fmt.Printf("   Quickmodes: internal: \"%s\"  heat pump: Dhw: \"%s\"  Zone: \"%s\"\n", ctrl.GetCurrentQuickMode(), dhwData.State.CurrentSpecialFunction, zoneData.State.CurrentSpecialFunction)
				fmt.Println("---------------------------------------------------------------------------------------------------------------------")
				lastPrint = time.Now()

			}
		}
	}

}
