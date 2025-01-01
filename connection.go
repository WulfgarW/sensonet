package sensonet

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/oauth2"
)

// Connection is the Sensonet connection
type Connection struct {
	client *http.Client
}

// NewConnection creates a new Sensonet device connection.
func NewConnection(ts oauth2.TokenSource, opts ...Option) (*Connection, error) {
	conn := &Connection{
		client: new(http.Client),
	}

	for _, opt := range opts {
		opt(conn)
	}

	conn.client.Transport = &oauth2.Transport{
		Source: ts,
		Base: &transport{
			conn.client.Transport,
		},
	}

	return conn, nil
}

func (c *Connection) GetHomes() ([]Home, error) {
	var res []Home
	url := API_URL_BASE + "/homes"
	req, _ := http.NewRequest("GET", url, nil)
	err := doJSON(c.client, req, &res)
	return res, err
}

func (c *Connection) GetSystem(systemId string) (*System, error) {
	var res System
	url := API_URL_BASE + fmt.Sprintf("/systems/%s/tli", systemId)
	req, _ := http.NewRequest("GET", url, nil)
	err := doJSON(c.client, req, &res)
	return &res, err
}

func (c *Connection) GetSystemDevices(systemId string) (*SystemDevices, error) {
	var res SystemDevices
	url := API_URL_BASE + fmt.Sprintf(DEVICES_URL, systemId)
	req, _ := http.NewRequest("GET", url, nil)
	err := doJSON(c.client, req, &res)
	return &res, err
}

func (c *Connection) StartZoneQuickVeto(systemId string, zone int, setpoint float32, duration float32) error {
	if zone < 0 {
		zone = ZONEINDEX_DEFAULT
	} // if parameter "zone" is negative, then the default value is used
	if setpoint < 0.0 {
		setpoint = ZONEVETOSETPOINT_DEFAULT
	} // if parameter "setpoint" is negative, then the default value is used
	if duration < 0.0 {
		duration = ZONEVETODURATION_DEFAULT
	} // if parameter "duration" is negative, then the default value is used
	url := API_URL_BASE + fmt.Sprintf(ZONEQUICKVETO_URL, systemId, zone)
	data := map[string]float32{
		"desiredRoomTemperatureSetpoint": setpoint,
		"duration":                       duration,
	}
	b, _ := json.Marshal(data)
	req, _ := http.NewRequest("POST", url, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")

	if _, err := doBody(c.client, req); err != nil {
		return fmt.Errorf("could not start quick veto: %w", err)
	}

	return nil
}

func (c *Connection) StopZoneQuickVeto(systemId string, zone int) error {
	if zone < 0 {
		zone = ZONEINDEX_DEFAULT
	} // if parameter "zone" is negative, then the default value is used

	url := API_URL_BASE + fmt.Sprintf(ZONEQUICKVETO_URL, systemId, zone)
	req, _ := http.NewRequest("DELETE", url, nil)

	_, err := doBody(c.client, req)
	return err
}

func (c *Connection) StartHotWaterBoost(systemId string, hotwaterIndex int) error {
	if hotwaterIndex < 0 {
		hotwaterIndex = HOTWATERINDEX_DEFAULT
	} // if parameter "hotwaterIndex" is negative, then the default value is used

	url := API_URL_BASE + fmt.Sprintf(HOTWATERBOOST_URL, systemId, hotwaterIndex)
	req, _ := http.NewRequest("POST", url, strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")

	_, err := doBody(c.client, req)
	return err
}

func (c *Connection) StopHotWaterBoost(systemId string, hotwaterIndex int) error {
	if hotwaterIndex < 0 {
		hotwaterIndex = HOTWATERINDEX_DEFAULT
	} // if parameter "hotwaterIndex" is negative, then the default value is used

	url := API_URL_BASE + fmt.Sprintf(HOTWATERBOOST_URL, systemId, hotwaterIndex)
	req, _ := http.NewRequest("DELETE", url, nil)

	_, err := doBody(c.client, req)
	return err
}

// // Returns the device data for give criteria
// func (c *Connection) GetDeviceData(devices []SystemDevices, whichDevices int) ([]DeviceAndInfo, error) {
// 	// var devices []DeviceAndInfo
// 	// systemDevices, err := c.getSystemDevices(systemid)
// 	// if err != nil {
// 	// 	return devices, fmt.Errorf("error getting sytem devices for %s: %w", systemid, err)
// 	// }
// 	var deviceAndInfo DeviceAndInfo
// 	if systemDevices.PrimaryHeatGenerator.DeviceUUID != "" && (whichDevices == DEVICES_PRIMARY_HEATER || whichDevices == DEVICES_ALL) {
// 		deviceAndInfo.Device = systemDevices.PrimaryHeatGenerator
// 		deviceAndInfo.Info = "primary_heat_generator"
// 		devices = append(devices, deviceAndInfo)
// 	}
// 	if whichDevices == DEVICES_SECONDARY_HEATER || whichDevices == DEVICES_ALL {
// 		for _, secHeatGen := range systemDevices.SecondaryHeatGenerators {
// 			deviceAndInfo.Device = secHeatGen
// 			deviceAndInfo.Info = "secondary_heat_generator"
// 			devices = append(devices, deviceAndInfo)
// 		}
// 	}

// 	if systemDevices.ElectricBackupHeater.DeviceUUID != "" && (whichDevices == DEVICES_BACKUP_HEATER || whichDevices == DEVICES_ALL) {
// 		deviceAndInfo.Device = systemDevices.ElectricBackupHeater
// 		deviceAndInfo.Info = "electric_backup_heater"
// 		devices = append(devices, deviceAndInfo)
// 	}
// 	return devices, nil
// }

// Returns the energy data systemId
func (c *Connection) GetEnergyData(systemid, deviceUuid, operationMode, energyType, resolution string, startDate, endDate time.Time) (EnergyData, error) {
	var energyData EnergyData
	v := url.Values{
		"resolution":    {resolution},
		"operationMode": {operationMode},
		"energyType":    {energyType},
		"startDate":     {startDate.Format("2006-01-02T15:04:05-07:00")},
		"endDate":       {endDate.Format("2006-01-02T15:04:05-07:00")},
	}

	url := API_URL_BASE + fmt.Sprintf(ENERGY_URL, systemid, deviceUuid) + v.Encode()
	req, _ := http.NewRequest("GET", url, nil)
	if err := doJSON(c.client, req, &energyData); err != nil {
		return energyData, fmt.Errorf("error getting energy data for %s: %w", deviceUuid, err)
	}
	return energyData, nil
}
