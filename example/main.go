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

	_ "github.com/joho/godotenv/autoload"
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

func main() {
	var (
		logger       = log.New(os.Stderr, "sensonet: ", log.Lshortfile)
		clientlogger = log.New(os.Stderr, "client: ", log.Lshortfile)
	)

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
	clientlogger.SetOutput(io.Discard) //comment this out, if logging in http client wanted
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

	fmt.Println("Fifth step: Using systemId of first element of homes[] and reading System structure from myVaillant portal")
	//We use the system ID of the first element (=index 0) of homes[]
	systemID := homes[0].SystemID
	state, err := conn.GetSystem(systemID)
	if err != nil {
		logger.Fatal(err)
	}

	fmt.Printf("   OutdoorTemperature: %.1f°C\n", state.State.System.OutdoorTemperature)
	for i, c := range state.State.Zones {
		fmt.Printf("   Zone %s: %.1f°C (Setpoint=%.1f°C)\n", state.Configuration.Zones[i].General.Name, c.CurrentRoomTemperature, c.DesiredRoomTemperatureSetpoint)
	}
	if len(state.State.DomesticHotWater)*len(state.Configuration.DomesticHotWater) > 0 {
		fmt.Printf("   HotWaterTemperature: %.1f°C (Setpoint=%.1f°C)\n", state.State.DomesticHotWater[0].CurrentDomesticHotWaterTemperature, state.Configuration.DomesticHotWater[0].TappingSetpoint)
	}
	if len(state.State.Dhw)*len(state.Configuration.Dhw) > 0 {
		fmt.Printf("   HotWaterTemperature (Dhw): %.1f°C (Setpoint=%.1f°C)\n", state.State.Dhw[0].CurrentDhwTemperature, state.Configuration.Dhw[0].TappingSetpoint)
	}

	/*fmt.Println("Next step: Starting zone quick veto")
	err = conn.StartZoneQuickVeto(systemID, 0, 18.0, 0.5)
	if err != nil {
		logger.Println(err)
	}*/

	/*fmt.Println("Next step: Stopping zone quick veto")
	err = conn.StopZoneQuickVeto(systemID, 0)
	if err != nil {
		logger.Println(err)
	}*/

	/*fmt.Println("Next step: Starting hotwater boost")
	err = conn.StartHotWaterBoost(systemID, -1)
	if err != nil {
		logger.Println(err)
	}*/

	/*fmt.Println("Next step: Stopping hotwater boost")
	err = conn.StopHotWaterBoost(systemID, -1)
	if err != nil {
		logger.Println(err)
	}*/

	var heatingPar sensonet.HeatingParStruct
	var hotwaterPar sensonet.HotwaterParStruct
	heatingPar.ZoneIndex = 0
	heatingPar.VetoSetpoint = 18.0
	heatingPar.VetoDuration = -1.0 //negative value means: use default
	hotwaterPar.Index = -1
	/*fmt.Println("Next step: Starting strategy based session")
	result, err := conn.StartStrategybased(systemID, sensonet.STRATEGY_HOTWATER_THEN_HEATING, &heatingPar, &hotwaterPar)
	if err != nil {
		logger.Println(err)
	} else {
		fmt.Printf("result=\"%s\"\n", result)
	}*/

	fmt.Println("Next step: Getting device data")
	devices, err := conn.GetDeviceData(systemID, sensonet.DEVICES_ALL)
	if err != nil {
		logger.Println(err)
	}
	fmt.Println("device data: \n ", devices)

	fmt.Println("Next step: Reading energy data")
	startDate, _ := time.Parse("2006-01-02 15:04:05MST", "2024-11-10 00:00:00CET")
	endDate, _ := time.Parse("2006-01-02 15:04:05MST", "2024-11-16 23:59:59CET")
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

	// Test, if the token refresh routine works as expected
	fmt.Println("Next step: Test, if the token refresh routine works as expected. Takes about 10 minutes")
	start := time.Now()
	for time.Now().Before(start.Add(10 * time.Minute)) {
		time.Sleep(60 * time.Second)
		state, err := conn.GetSystem(systemID)
		if err != nil {
			logger.Fatal(err)
		}
		fmt.Println("   It is now:", time.Now())
		dhwData := sensonet.GetDhwData(state, -1)
		zoneData := sensonet.GetZoneData(state, heatingPar.ZoneIndex)
		fmt.Printf("   Quickmodes: internal: \"%s\"  heat pump: Dhw: \"%s\"  Zone: \"%s\"\n", conn.GetCurrentQuickMode(), dhwData.State.CurrentSpecialFunction, zoneData.State.CurrentSpecialFunction)
	}

	fmt.Println("Next step: Stopping strategy based session")
	result2, err := conn.StopStrategybased(systemID, sensonet.STRATEGY_HOTWATER_THEN_HEATING, &heatingPar, &hotwaterPar)
	if err != nil {
		logger.Println(err)
	} else {
		fmt.Printf("result=\"%s\"\n", result2)
	}

}
