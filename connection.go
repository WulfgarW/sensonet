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
func NewConnection(ts oauth2.TokenSource, opts ...ConnOption) (*Connection, error) {
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

// Returns all "homes" that belong to the current user under the myVaillant portal
func (c *Connection) GetHomes() (Homes, error) {
	var res Homes
	url := API_URL_BASE + "/homes"
	req, _ := http.NewRequest("GET", url, nil)
	err := doJSON(c.client, req, &res)
	return res, err
}

// Returns the system report (state, properties and configuration) for a specific systemId
func (c *Connection) GetSystem(systemId string) (SystemStatus, error) {
	var state SystemStatus
	url := API_URL_BASE + fmt.Sprintf(SYSTEMS_URL, systemId)
	req, _ := http.NewRequest("GET", url, nil)
	err := doJSON(c.client, req, &state)
	return state, err
}

// Returns the system devices for a specific systemId
func (c *Connection) GetSystemDevices(systemId string) (SystemDevices, error) {
	var systemDevices SystemDevices
	url := API_URL_BASE + fmt.Sprintf(DEVICES_URL, systemId)
	req, _ := http.NewRequest("GET", url, nil)
	err := doJSON(c.client, req, &systemDevices)
	return systemDevices, err
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
		return err
	}
	return nil
}

func (c *Connection) StopZoneQuickVeto(systemId string, zone int) error {
	if zone < 0 {
		zone = ZONEINDEX_DEFAULT
	} // if parameter "zone" is negative, then the default value is used

	url := API_URL_BASE + fmt.Sprintf(ZONEQUICKVETO_URL, systemId, zone)
	req, _ := http.NewRequest("DELETE", url, nil)

	if _, err := doBody(c.client, req); err != nil {
		return err
	}
	return nil
}

func (c *Connection) StartHotWaterBoost(systemId string, hotwaterIndex int) error {
	if hotwaterIndex < 0 {
		hotwaterIndex = HOTWATERINDEX_DEFAULT
	} // if parameter "hotwaterIndex" is negative, then the default value is used

	url := API_URL_BASE + fmt.Sprintf(HOTWATERBOOST_URL, systemId, hotwaterIndex)
	req, _ := http.NewRequest("POST", url, strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")

	if _, err := doBody(c.client, req); err != nil {
		return err
	}
	return nil
}

func (c *Connection) StopHotWaterBoost(systemId string, hotwaterIndex int) error {
	if hotwaterIndex < 0 {
		hotwaterIndex = HOTWATERINDEX_DEFAULT
	} // if parameter "hotwaterIndex" is negative, then the default value is used

	url := API_URL_BASE + fmt.Sprintf(HOTWATERBOOST_URL, systemId, hotwaterIndex)
	req, _ := http.NewRequest("DELETE", url, nil)

	if _, err := doBody(c.client, req); err != nil {
		return err
	}
	return nil
}

// Returns the device data for given criteria
func (c *Connection) GetDeviceData(systemId string, whichDevices int) ([]DeviceAndInfo, error) {
	var devices []DeviceAndInfo
	systemDevices, err := c.GetSystemDevices(systemId)
	if err != nil {
		return devices, err
	}
	var deviceAndInfo DeviceAndInfo
	if systemDevices.PrimaryHeatGenerator.DeviceUUID != "" && (whichDevices == DEVICES_PRIMARY_HEATER || whichDevices == DEVICES_ALL) {
		deviceAndInfo.Device = systemDevices.PrimaryHeatGenerator
		deviceAndInfo.Info = "primary_heat_generator"
		devices = append(devices, deviceAndInfo)
	}
	if whichDevices == DEVICES_SECONDARY_HEATER || whichDevices == DEVICES_ALL {
		for _, secHeatGen := range systemDevices.SecondaryHeatGenerators {
			deviceAndInfo.Device = secHeatGen
			deviceAndInfo.Info = "secondary_heat_generator"
			devices = append(devices, deviceAndInfo)
		}
	}

	if systemDevices.ElectricBackupHeater.DeviceUUID != "" && (whichDevices == DEVICES_BACKUP_HEATER || whichDevices == DEVICES_ALL) {
		deviceAndInfo.Device = systemDevices.ElectricBackupHeater
		deviceAndInfo.Info = "electric_backup_heater"
		devices = append(devices, deviceAndInfo)
	}
	return devices, nil
}

// Returns the energy data for systemId, deviceUuid and other given criteria
func (c *Connection) GetEnergyData(systemId, deviceUuid, operationMode, energyType, resolution string, startDate, endDate time.Time) (EnergyData, error) {
	var energyData EnergyData
	v := url.Values{
		"resolution":    {resolution},
		"operationMode": {operationMode},
		"energyType":    {energyType},
		"startDate":     {startDate.Format("2006-01-02T15:04:05-07:00")},
		"endDate":       {endDate.Format("2006-01-02T15:04:05-07:00")},
	}

	url := API_URL_BASE + fmt.Sprintf(ENERGY_URL, systemId, deviceUuid) + v.Encode()
	req, _ := http.NewRequest("GET", url, nil)
	if err := doJSON(c.client, req, &energyData); err != nil {
		return energyData, err
	}
	return energyData, nil
}

// Returns the mpc data for systemId
func (c *Connection) GetMpcData(systemId string) (MpcData, error) {
	var mpcData MpcData

	url := API_URL_BASE + fmt.Sprintf(MPC_URL, systemId)
	req, _ := http.NewRequest("GET", url, nil)
	if err := doJSON(c.client, req, &mpcData); err != nil {
		return mpcData, err
	}
	return mpcData, nil
}

// Returns the current power consumption for systemId
func (c *Connection) GetSystemCurrentPower(systemId string) (float64, error) {
	mpcData, err := c.GetMpcData(systemId)
	if err != nil || len(mpcData.Devices) < 1 {
		return -1.0, err
	}
	totalPower := 0.0
	for _, dev := range mpcData.Devices {
		totalPower = totalPower + dev.CurrentPower
	}
	return totalPower, nil
}

// Returns the current power consumption and product name for deviceUuid. If "All" is given as deviceUuid, then the function return the power consumption and product name for all devices of systemId
func (c *Connection) GetDeviceCurrentPower(systemId, deviceUuid string) (DevicePowerMap, error) {
	devicePowerMap := make(DevicePowerMap)
	if deviceUuid == "All" {
		devicePowerMap["All"] = DevicePower{CurrentPower: -1.0, ProductName: "All Devices"}
	}
	mpcData, err := c.GetMpcData(systemId)
	if err != nil || len(mpcData.Devices) < 1 {
		return devicePowerMap, err
	}
	devices, err := c.GetDeviceData(systemId, DEVICES_ALL)
	if err != nil {
		return devicePowerMap, err
	}
	totalPower := 0.0
	for _, dev := range mpcData.Devices {
		totalPower = totalPower + dev.CurrentPower
		if dev.DeviceID == deviceUuid || deviceUuid == "All" {
			for _, dev2 := range devices {
				if dev.DeviceID == dev2.Device.DeviceUUID {
					devicePowerMap[deviceUuid] = DevicePower{CurrentPower: dev.CurrentPower, ProductName: dev2.Device.ProductName}
				}
			}
		}
	}
	devicePowerMap["All"] = DevicePower{CurrentPower: totalPower, ProductName: "All Devices"}
	return devicePowerMap, nil
}
