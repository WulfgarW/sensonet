package sensonet

import (
	"bytes"
	"fmt"
	"math"
	"net/http"
	"time"

	"github.com/evcc-io/evcc/util/request"
)

type Switch struct {
	*Connection
}

func NewSwitch(conn *Connection) *Switch {
	res := &Switch{
		Connection: conn,
	}

	return res
}

// Enabled implements the api.Charger interface
func (sh *Switch) Enabled() (bool, error) {
	var err error
	d := sh.Connection
	// If token expires in less than 3 minutes, a token refresh is called
	if time.Now().Add(time.Duration(3 * int64(time.Minute))).After(d.tokenExpiresAt) {
		err = d.refreshToken()
		if err != nil {
			err = fmt.Errorf("could not refresh token. error: %s", err)
			return false, err
		}
		//d.log.DEBUG.Println("Refresh token successful")
		d.tokenExpiresAt = time.Now().Add(time.Duration(d.tokenRes.ExpiresIn * int64(time.Second)))
		d.log.DEBUG.Printf("Refreshed token expires at: %02d:%02d:%02d", d.tokenExpiresAt.Hour(), d.tokenExpiresAt.Minute(), d.tokenExpiresAt.Second())
	}

	res, err := d.statusCache.Get()
	if err != nil {
		d.log.ERROR.Println("Switch.Enabled. Error: ", err)
		return false, err
	}
	d.log.DEBUG.Println("Status last read from myVaillant portal at:", time.Unix(res.Timestamp, 0))
	if d.currentQuickmode != "" {
		d.log.DEBUG.Println("In Switch.Enabled: Connection.currentQuickmode:", d.currentQuickmode, "started at:", (d.quickmodeStarted).Format("2006-01-02 15:04:05"))
	} else {
		d.log.DEBUG.Println("In Switch.Enabled: Connection.currentQuickmode not set. Timestamp:", (d.quickmodeStarted).Format("2006-01-02 15:04:05"))
		if (res.Hotwater.CurrentQuickmode != "") && (res.Hotwater.CurrentQuickmode != "REGULAR") {
			d.log.DEBUG.Println("In Switch.Enabled: res.Hotwater.CurrentQuickmode should be inactive but is on")
			if d.quickmodeStarted.Add(5 * time.Minute).Before(time.Now()) {
				// When the reported hotwater.CurrentQuickmode is not "Regular" more then 5 minutes after the end of the charge session (or the start of evcc),
				// this means that the heat pump is in hotwater boost
				d.currentQuickmode = QUICKMODE_HOTWATER
				d.quickmodeStarted = time.Now()
				d.onoff = true
			}
		}
		for _, z := range res.Zones {
			if z.Index == d.heatingZone {
				d.log.DEBUG.Println("In Switch.Enabled: Zone quick mode:", z.CurrentQuickmode, ", Temperature Setpoint:", z.QuickVeto.TemperatureSetpoint, "(", d.quickVetoSetPoint, "), Expires at:", d.quickVetoExpiresAt)
				if (z.CurrentQuickmode != "") && (z.CurrentQuickmode != "NONE") {
					d.log.DEBUG.Println("In Switch.Enabled: z.CurrentQuickmode should be inactive but is on")
					if d.quickmodeStarted.Add(5 * time.Minute).Before(time.Now()) {
						// When the reported z.CurrentQuickmode is not "NONE" more then 5 minutes after the end of a charge session (or the start of evcc),
						// this means that the zone quick veto startet by other means as evcc
						d.currentQuickmode = QUICKMODE_HEATING
						d.quickmodeStarted = time.Now()
						d.onoff = true
					}
				}
			}
		}
	}
	switch d.currentQuickmode {
	case QUICKMODE_HOTWATER:
		d.log.DEBUG.Println("In Switch.Enabled: Hotwater quick mode:", res.Hotwater.CurrentQuickmode)
		if (res.Hotwater.CurrentQuickmode == "") || (res.Hotwater.CurrentQuickmode == "REGULAR") {
			d.log.DEBUG.Println("In Switch.Enabled: res.Hotwater.CurrentQuickmode should be active but is off")
			if d.quickmodeStarted.Add(5 * time.Minute).Before(time.Now()) {
				// When the reported hotwater.CurrentQuickmode has changed to "Regular" more then 5 minutes after the beginning of the charge session,
				// this means that the heat pump has stopped the hotwater boost itself
				d.currentQuickmode = ""
				d.quickmodeStopped = time.Now()
				d.onoff = false
			}
		}
	case QUICKMODE_HEATING:
		for _, z := range res.Zones {
			if z.Index == d.heatingZone {
				d.log.DEBUG.Println("In Switch.Enabled: Zone quick mode:", z.CurrentQuickmode, ", Temperature Setpoint:", z.QuickVeto.TemperatureSetpoint, "(", d.quickVetoSetPoint, "), Expires at:", d.quickVetoExpiresAt)
				if (z.CurrentQuickmode == "") || (z.CurrentQuickmode == "NONE") {
					d.log.DEBUG.Println("In Switch.Enabled: z.CurrentQuickmode should be active but is off")
					if d.quickmodeStarted.Add(5 * time.Minute).Before(time.Now()) {
						// When the reported z.CurrentQuickmode has changed to "NONE" more then 5 minutes after the beginning of the charge session,
						// this means that the zone quick veto ended or was stopped by other means as evcc
						d.currentQuickmode = ""
						d.quickmodeStopped = time.Now()
						d.onoff = false
					}
				}
			}
		}
	case QUICKMODE_NOTHING:
		if d.quickmodeStarted.Add(10 * time.Minute).Before(time.Now()) {
			d.log.DEBUG.Println("Idle charge mode for more than 10 minutes. Turning it off")
			d.currentQuickmode = ""
			d.quickmodeStopped = time.Now()
			d.onoff = false
		}
	case "":
		//Nothing to do
	default:
		d.log.ERROR.Println("Unknown quick mode in case statement:", d.currentQuickmode)
	}
	return d.onoff, nil
}

// Enable implements the api.Charger interface
func (sh *Switch) Enable(enable bool) error {
	d := sh.Connection
	//Reset status cache and get new reports from sensonet before starting or stopping quick modes
	d.reset()
	res, err := d.statusCache.Get()
	if err != nil {
		err = fmt.Errorf("could not read status cache before hotwater boost: %s", err)
		return err
	}
	if d.currentQuickmode == "" && d.quickmodeStopped.After(d.quickmodeStarted) && d.quickmodeStopped.Add(2*time.Minute).After(time.Now()) {
		enable = false
	}
	if enable {
		whichQuickMode, err := d.WhichQuickMode()
		if err != nil {
			err = fmt.Errorf("error while computing which quick mode to start: %s", err)
			return err
		}

		switch whichQuickMode {
		case 1:
			err = sh.startHotWaterBoost(&res)
			if err == nil {
				d.currentQuickmode = QUICKMODE_HOTWATER
				d.quickmodeStarted = time.Now()
				d.log.DEBUG.Println("Starting quick mode (hotwater boost)", res.Hotwater.CurrentQuickmode)
			}
		case 2:
			err = sh.startZoneQuickVeto(&res)
			if err == nil {
				d.currentQuickmode = QUICKMODE_HEATING
				d.quickmodeStarted = time.Now()
				d.log.DEBUG.Println("Starting zone quick veto")
			}
		default:
			if d.currentQuickmode == QUICKMODE_HOTWATER {
				//if hotwater boost active, then stop it
				err = sh.stopHotWaterBoost(&res)
				if err == nil {
					d.log.DEBUG.Println("Stopping Hotwater Boost")
				}
			}
			if d.currentQuickmode == QUICKMODE_HEATING {
				//if zone quick veto active, then stop it
				err = sh.stopZoneQuickVeto()
				if err == nil {
					d.log.DEBUG.Println("Stopping Zone Quick Veto")
				}
			}
			d.currentQuickmode = QUICKMODE_NOTHING
			d.quickmodeStarted = time.Now()
			d.log.DEBUG.Println("Enable called but no quick mode possible. Starting idle mode")
			//d.log.INFO.Println("Enable called but no quick mode possible")
		}
	} else {
		switch d.currentQuickmode {
		case QUICKMODE_HOTWATER:
			err = sh.stopHotWaterBoost(&res)
			if err == nil {
				d.log.DEBUG.Println("Stopping Quick Mode", res.Hotwater.CurrentQuickmode)
			}
		case QUICKMODE_HEATING:
			err = sh.stopZoneQuickVeto()
			if err == nil {
				d.log.DEBUG.Println("Stopping Zone Quick Veto")
			}
		case QUICKMODE_NOTHING:
			d.log.DEBUG.Println("Stopping idle quick mode")
		default:
			d.log.DEBUG.Println("Nothing to do, no quick mode active")
		}
		d.currentQuickmode = ""
		d.quickmodeStarted = time.Now()
	}
	//Reset status cache and get new reports from sensonet
	/*d.reset()
	res, err1 := d.statusCache.Get()
	if err1 != nil {
		err1 = fmt.Errorf("could not get current live and system report after starting/stopping quick modes: %s", err1)
		return err1
	}*/
	if err == nil {
		d.onoff = enable
	}
	return err
}

// CurrentPower implements the api.Meter interface
// Those are just dummy values. For eal values, an energy meter like Shelly 3EM is necessary
func (sh *Switch) CurrentPower() (float64, error) {
	var power float64

	d := sh.Connection
	d.log.DEBUG.Println("Switch.CurrentPower", d.currentQuickmode, d.quickmodeStarted.Format("2006-01-02 15:04:05"))

	// Returns dummy values for CurrentPower if called
	if d.onoff {
		power = 3000.0
	} else {
		power = 0.0
	}
	if d.currentQuickmode == QUICKMODE_HEATING {
		power = 1500.0
	}
	return power, nil
}

func (sh *Switch) startHotWaterBoost(relData *Vr921RelevantDataStruct) error {
	c := sh.Connection
	urlHotwaterBoost := API_URL_BASE + fmt.Sprintf(HOTWATERBOOST_URL, c.systemId, relData.Hotwater.Index)
	req, err := http.NewRequest("POST", urlHotwaterBoost, request.MarshalJSON(map[string]string{}))
	if err != nil {
		err = fmt.Errorf("client: could not create request: %s", err)
		return err
	}
	req.Header = c.getSensonetHttpHeader()
	req.Header.Set("Content-Type", "application/json")
	var resp []byte
	resp, err = c.DoBody(req)
	if err != nil {
		err = fmt.Errorf("could not start hotwater boost. Error: %s", err)
		c.log.DEBUG.Printf("Response: %s\n", resp)
		return err
	}
	return err
}

func (sh *Switch) stopHotWaterBoost(relData *Vr921RelevantDataStruct) error {
	c := sh.Connection
	urlHotwaterBoost := API_URL_BASE + fmt.Sprintf(HOTWATERBOOST_URL, c.systemId, relData.Hotwater.Index)
	req, err := http.NewRequest("DELETE", urlHotwaterBoost, bytes.NewBuffer(nil))
	if err != nil {
		err = fmt.Errorf("client: could not create request: %s", err)
		return err
	}
	req.Header = c.getSensonetHttpHeader()
	var resp []byte
	resp, err = c.DoBody(req)
	if err != nil {
		err = fmt.Errorf("could not stop hotwater boost. Error: %s", err)
		c.log.DEBUG.Printf("Response: %s\n", resp)
		return err
	}
	return err
}

func (sh *Switch) startZoneQuickVeto(relData *Vr921RelevantDataStruct) error {
	c := sh.Connection
	urlZoneQuickVeto := API_URL_BASE + fmt.Sprintf(ZONEQUICKVETO_URL, c.systemId, c.heatingZone)
	temperatureSetpoint := 0.0
	for _, z := range relData.Zones {
		if z.Index == c.heatingZone {
			temperatureSetpoint = z.CurrentDesiredSetpoint
		}
	}
	if temperatureSetpoint == 0.0 {
		c.log.ERROR.Printf("Could not detect current desired setpoint for zone: %01d. Probably inactive due to time program.\n", c.heatingZone)
		temperatureSetpoint = 20.0
	}
	// Temperature Setpoint for quick veto is rounded to 0.5 °C
	vetoSetpoint := float32(math.Round(2*(temperatureSetpoint+c.heatingTemperatureOffset)) / 2.0)
	vetoDuration := float32(0.5)
	data := map[string]float32{
		"desiredRoomTemperatureSetpoint": float32(vetoSetpoint),
		"duration":                       vetoDuration, // duration for quick veto is 0,5 hours
		//		"duration":             float32(c.heatingVetoDuration) / 30.0,
	}
	req, err := http.NewRequest("POST", urlZoneQuickVeto, request.MarshalJSON(data))
	if err != nil {
		err = fmt.Errorf("client: could not create request: %s", err)
		return err
	}
	req.Header = c.getSensonetHttpHeader()
	req.Header.Set("Content-Type", "application/json")
	var resp []byte
	c.log.DEBUG.Printf("Sending POST request to: %s\n", urlZoneQuickVeto)
	resp, err = c.DoBody(req)
	if err != nil {
		err = fmt.Errorf("could not start quick veto. Error: %s", err)
		c.log.DEBUG.Printf("Response: %s\n", resp)
		return err
	}
	c.quickVetoSetPoint = vetoSetpoint
	c.quickVetoExpiresAt = (time.Now().Add(time.Duration(int64(vetoDuration*60) * int64(time.Minute)))).Format("15:04")
	return err
}

func (sh *Switch) stopZoneQuickVeto() error {
	c := sh.Connection
	urlZoneQuickVeto := API_URL_BASE + fmt.Sprintf(ZONEQUICKVETO_URL, c.systemId, c.heatingZone)
	req, err := http.NewRequest("DELETE", urlZoneQuickVeto, bytes.NewBuffer(nil))
	if err != nil {
		err = fmt.Errorf("client: could not create request: %s", err)
		return err
	}
	req.Header = c.getSensonetHttpHeader()
	var resp []byte
	c.log.DEBUG.Printf("Sending DELETE request to: %s\n", urlZoneQuickVeto)
	resp, err = c.DoBody(req)
	if err != nil {
		err = fmt.Errorf("could not stop quick veto. Error: %s", err)
		c.log.DEBUG.Printf("Response: %s\n", resp)
		return err
	}
	c.quickVetoSetPoint = 0
	c.quickVetoExpiresAt = ""
	return err
}
