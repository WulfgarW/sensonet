package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/WulfgarW/sensonet/sensonet"
	_ "github.com/joho/godotenv/autoload"
	"golang.org/x/oauth2"
)

const TOKEN_FILE = ".sensonet-token.json"
const CREDENTIALS_FILE = ".sensonet-credentials.json"

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

func main() {
	var (
		logger = log.New(os.Stderr, "sensonet: ", log.Lshortfile)
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
	// Creates an http client and opens the connection to the myVaillant portal.
	// If a token is provided, then it's validity is checked and a token refresh called if necessary.
	// If no token is provided or if a normal refresh is not possible, a login using the credentials is done
	conn, newToken, err := sensonet.NewConnection(logger, credentials, token)
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
	for i, c := range state.State.Circuits {
		fmt.Printf("   Zone %s: %.1f°C (Setpoint=%.1f°C)\n", state.Configuration.Zones[i].General.Name, c.CurrentCircuitFlowTemperature, c.HeatingCircuitFlowSetpoint)
	}
	if len(state.State.DomesticHotWater)*len(state.Configuration.DomesticHotWater) > 0 {
		fmt.Printf("   HotWaterTemperature: %.1f°C (Setpoint=%.1f°C)\n", state.State.DomesticHotWater[0].CurrentDomesticHotWaterTemperature, state.Configuration.DomesticHotWater[0].TappingSetpoint)
	}
	if len(state.State.Dhw)*len(state.Configuration.Dhw) > 0 {
		fmt.Printf("   HotWaterTemperature (Dhw): %.1f°C (Setpoint=%.1f°C)\n", state.State.Dhw[0].CurrentDhwTemperature, state.Configuration.Dhw[0].TappingSetpoint)
	}

	/*fmt.Println("Next step: Starting zone quick veto")
	err = conn.StartZoneQuickVeto(systemID, 0, 19.0, 0.5)
	if err != nil {
		logger.Println(err)
	}*/

	fmt.Println("Next step: Stopping zone quick veto")
	err = conn.StopZoneQuickVeto(systemID, 0)
	if err != nil {
		logger.Println(err)
	}

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

	// Test, if the token refresh routine works as expected
	fmt.Println("Next step: Test, if the token refresh routine works as expected. Takes about 15 minutes")
	start := time.Now()
	for time.Now().Before(start.Add(15 * time.Minute)) {
		time.Sleep(time.Minute)
		state, err := conn.GetSystem(systemID)
		if err != nil {
			logger.Fatal(err)
		}
		fmt.Println("   It is now:", time.Now())
		fmt.Printf("   OutdoorTemperature: %.1f°C\n", state.State.System.OutdoorTemperature)
	}

}
