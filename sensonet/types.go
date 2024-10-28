package sensonet

import (
	"time"

	"golang.org/x/oauth2"
)

const (
	CLIENT_ID = "myvaillant"

	AUTH_BASE_URL = "https://identity.vaillant-group.com/auth/realms"
	LOGIN_URL     = AUTH_BASE_URL + "/%s/login-actions/authenticate"
	TOKEN_URL     = AUTH_BASE_URL + "/%s/protocol/openid-connect/token"
	AUTH_URL      = AUTH_BASE_URL + "/%s/protocol/openid-connect/auth"
	API_URL_BASE  = "https://api.vaillant-group.com/service-connected-control/end-user-app-api/v1"
)

const (
	HOTWATERBOOST_URL = "/systems/%s/tli/domestic-hot-water/%01d/boost"
	ZONEQUICKVETO_URL = "/systems/%s/tli/zones/%01d/quick-veto"

	// SYSTEM_URL     = "/systemcontrol/tli/v1"
	// FACILITIES_URL = "not to be used"
	// LIVEREPORT_URL = "/livereport/v1"
	HOTWATERINDEX_DEFAULT = 255
)

type CredentialsStruct struct {
	User     string `json:"user"`
	Password string `json:"password"`
	Realm    string `json:"realm"`
}

type TokenRequestStruct struct {
	oauth2.Token
	RefreshExpiresIn int64  `json:"refresh_expires_in"`
	NotBeforePolicy  int    `json:"not-before-policy"`
	SessionState     string `json:"session_state"`
	Scope            string `json:"scope"`
}

type Homes []struct {
	HomeName string `json:"homeName"`
	Address  struct {
		Street      string `json:"street"`
		Extension   any    `json:"extension"`
		City        string `json:"city"`
		PostalCode  string `json:"postalCode"`
		CountryCode string `json:"countryCode"`
	} `json:"address"`
	SerialNumber    string `json:"serialNumber"`
	SystemID        string `json:"systemId"`
	ProductMetadata struct {
		ProductType    string `json:"productType"`
		ProductionYear string `json:"productionYear"`
		ProductionWeek string `json:"productionWeek"`
		ArticleNumber  string `json:"articleNumber"`
	} `json:"productMetadata"`
	State               string    `json:"state"`
	MigrationState      string    `json:"migrationState"`
	MigrationFinishedAt time.Time `json:"migrationFinishedAt"`
	OnlineState         string    `json:"onlineState"`
	Firmware            struct {
		Version        string `json:"version"`
		UpdateEnabled  bool   `json:"updateEnabled"`
		UpdateRequired bool   `json:"updateRequired"`
	} `json:"firmware"`
	Nomenclature       string `json:"nomenclature"`
	Cag                bool   `json:"cag"`
	CountryCode        string `json:"countryCode"`
	ProductInformation string `json:"productInformation"`
	FirmwareVersion    string `json:"firmwareVersion"`
}

type SystemStatus struct {
	State struct {
		System struct {
			OutdoorTemperature           float64 `json:"outdoorTemperature"`
			OutdoorTemperatureAverage24H float64 `json:"outdoorTemperatureAverage24h"`
			SystemFlowTemperature        float64 `json:"systemFlowTemperature"`
			SystemWaterPressure          float64 `json:"systemWaterPressure"`
			EnergyManagerState           string  `json:"energyManagerState"`
			SystemOff                    bool    `json:"systemOff"`
		} `json:"system"`
		Zones            []StateZone             `json:"zones"`
		Circuits         []StateCircuit          `json:"circuits"`
		Dhw              []StateDhw              `json:"dhw"`
		DomesticHotWater []StateDomesticHotWater `json:"domesticHotWater"`
		// Ventilations
	} `json:"state"`
	Properties struct {
		System struct {
			ControllerType                     string  `json:"controllerType"`
			SystemScheme                       int     `json:"systemScheme"`
			BackupHeaterType                   string  `json:"backupHeaterType"`
			BackupHeaterAllowedFor             string  `json:"backupHeaterAllowedFor"`
			ModuleConfigurationVR71            int     `json:"moduleConfigurationVR71"`
			EnergyProvidePowerCutBehavior      string  `json:"energyProvidePowerCutBehavior"`
			SmartPhotovoltaicBufferOffset      float64 `json:"smartPhotovoltaicBufferOffset"`
			ExternalEnergyManagementActivation bool    `json:"externalEnergyManagementActivation"`
		} `json:"system"`
		Zones            []PropertiesZone             `json:"zones"`
		Circuits         []PropertiesCircuit          `json:"circuits"`
		Dhw              []PropertiesDhw              `json:"dhw"`
		DomesticHotWater []PropertiesDomesticHotWater `json:"domesticHotWater"`
		// Ventilations
	} `json:"properties"`
	Configuration struct {
		System struct {
			ContinuousHeatingStartSetpoint float64 `json:"continuousHeatingStartSetpoint"`
			AlternativePoint               float64 `json:"alternativePoint"`
			HeatingCircuitBivalencePoint   float64 `json:"heatingCircuitBivalencePoint"`
			DhwBivalencePoint              float64 `json:"dhwBivalencePoint"`
			AdaptiveHeatingCurve           bool    `json:"adaptiveHeatingCurve"`
			DhwMaximumLoadingTime          int     `json:"dhwMaximumLoadingTime"`
			DhwHysteresis                  float64 `json:"dhwHysteresis"`
			DhwFlowSetpointOffset          float64 `json:"dhwFlowSetpointOffset"`
			ContinuousHeatingRoomSetpoint  float64 `json:"continuousHeatingRoomSetpoint"`
			HybridControlStrategy          string  `json:"hybridControlStrategy"`
			MaxFlowSetpointHpError         float64 `json:"maxFlowSetpointHpError"`
			DhwMaximumTemperature          float64 `json:"dhwMaximumTemperature"`
			MaximumPreheatingTime          int     `json:"maximumPreheatingTime"`
			ParalellTankLoadingAllowed     bool    `json:"paralellTankLoadingAllowed"`
		} `json:"system"`
		Zones            []ConfigurationZone             `json:"zones"`
		Circuits         []ConfigurationCircuit          `json:"circuits"`
		Dhw              []ConfigurationDhw              `json:"dhw"`
		DomesticHotWater []ConfigurationDomesticHotWater `json:"domesticHotWater"`
		// Ventilations
	} `json:"configuration"`
}

type StateZone struct {
	Index                                 int     `json:"index"`
	DesiredRoomTemperatureSetpointHeating float64 `json:"desiredRoomTemperatureSetpointHeating"`
	DesiredRoomTemperatureSetpoint        float64 `json:"desiredRoomTemperatureSetpoint"`
	CurrentRoomTemperature                float64 `json:"currentRoomTemperature,omitempty"`
	CurrentRoomHumidity                   float64 `json:"currentRoomHumidity,omitempty"`
	CurrentSpecialFunction                string  `json:"currentSpecialFunction"`
	HeatingState                          string  `json:"heatingState"`
}

type StateCircuit struct {
	Index                         int     `json:"index"`
	CircuitState                  string  `json:"circuitState"`
	CurrentCircuitFlowTemperature float64 `json:"currentCircuitFlowTemperature,omitempty"`
	HeatingCircuitFlowSetpoint    float64 `json:"heatingCircuitFlowSetpoint"`
	CalculatedEnergyManagerState  string  `json:"calculatedEnergyManagerState"`
}

type StateDhw struct {
	Index                  int     `json:"index"`
	CurrentSpecialFunction string  `json:"currentSpecialFunction"`
	CurrentDhwTemperature  float64 `json:"currentDhwTemperature"`
}

type StateDomesticHotWater struct {
	Index                              int     `json:"index"`
	CurrentSpecialFunction             string  `json:"currentSpecialFunction"`
	CurrentDomesticHotWaterTemperature float64 `json:"currentDomesticHotWaterTemperature"`
}

type PropertiesZone struct {
	Index                  int    `json:"index"`
	IsActive               bool   `json:"isActive"`
	ZoneBinding            string `json:"zoneBinding"`
	IsCoolingAllowed       bool   `json:"isCoolingAllowed"`
	AssociatedCircuitIndex int    `json:"associatedCircuitIndex"`
}

type PropertiesCircuit struct {
	Index                    int    `json:"index"`
	MixerCircuitTypeExternal string `json:"mixerCircuitTypeExternal"`
	HeatingCircuitType       string `json:"heatingCircuitType"`
}

type PropertiesDhw struct {
	Index       int     `json:"index"`
	MinSetpoint float64 `json:"minSetpoint"`
	MaxSetpoint float64 `json:"maxSetpoint"`
}

type PropertiesDomesticHotWater struct {
	Index       int     `json:"index"`
	MinSetpoint float64 `json:"minSetpoint"`
	MaxSetpoint float64 `json:"maxSetpoint"`
}

type TimeSlot struct {
	StartTime int `json:"startTime"`
	EndTime   int `json:"endTime"`
}

type Setpoint struct {
	StartTime int     `json:"startTime"`
	EndTime   int     `json:"endTime"`
	Setpoint  float64 `json:"setpoint"`
}

type MetaInfo struct {
	MinSlotsPerDay          int  `json:"minSlotsPerDay"`
	MaxSlotsPerDay          int  `json:"maxSlotsPerDay"`
	SetpointRequiredPerSlot bool `json:"setpointRequiredPerSlot"`
}

type TimeProgram struct {
	MetaInfo  MetaInfo   `json:"metaInfo"`
	Monday    []Setpoint `json:"monday"`
	Tuesday   []Setpoint `json:"tuesday"`
	Wednesday []Setpoint `json:"wednesday"`
	Thursday  []Setpoint `json:"thursday"`
	Friday    []Setpoint `json:"friday"`
	Saturday  []Setpoint `json:"saturday"`
	Sunday    []Setpoint `json:"sunday"`
}

type ConfigurationZone struct {
	Index   int `json:"index"`
	General struct {
		Name                 string    `json:"name"`
		HolidayStartDateTime time.Time `json:"holidayStartDateTime"`
		HolidayEndDateTime   time.Time `json:"holidayEndDateTime"`
		HolidaySetpoint      float64   `json:"holidaySetpoint"`
	} `json:"general"`
	Heating struct {
		OperationModeHeating      string      `json:"operationModeHeating"`
		SetBackTemperature        float64     `json:"setBackTemperature"`
		ManualModeSetpointHeating float64     `json:"manualModeSetpointHeating"`
		TimeProgramHeating        TimeProgram `json:"timeProgramHeating"`
	} `json:"heating"`
}

type ConfigurationCircuit struct {
	Index                                  int     `json:"index"`
	HeatingCurve                           float64 `json:"heatingCurve"`
	HeatingFlowTemperatureMinimumSetpoint  float64 `json:"heatingFlowTemperatureMinimumSetpoint"`
	HeatingFlowTemperatureMaximumSetpoint  float64 `json:"heatingFlowTemperatureMaximumSetpoint"`
	HeatDemandLimitedByOutsideTemperature  float64 `json:"heatDemandLimitedByOutsideTemperature"`
	HeatingCircuitFlowSetpointExcessOffset float64 `json:"heatingCircuitFlowSetpointExcessOffset"`
	SetBackModeEnabled                     bool    `json:"setBackModeEnabled"`
	RoomTemperatureControlMode             string  `json:"roomTemperatureControlMode"`
}

type ConfigurationDhw struct {
	Index                      int         `json:"index"`
	OperationModeDhw           string      `json:"operationModeDhw"`
	TappingSetpoint            float64     `json:"tappingSetpoint"`
	HolidayStartDateTime       time.Time   `json:"holidayStartDateTime"`
	HolidayEndDateTime         time.Time   `json:"holidayEndDateTime"`
	TimeProgramDhw             TimeProgram `json:"timeProgramDhw"`
	TimeProgramCirculationPump TimeProgram `json:"timeProgramCirculationPump"`
}

type ConfigurationDomesticHotWater struct {
	Index                         int         `json:"index"`
	OperationModeDomesticHotWater string      `json:"operationModeDomesticHotWater"`
	TappingSetpoint               float64     `json:"tappingSetpoint"`
	HolidayStartDateTime          time.Time   `json:"holidayStartDateTime"`
	HolidayEndDateTime            time.Time   `json:"holidayEndDateTime"`
	TimeProgramDomesticHotWater   TimeProgram `json:"timeProgramDomesticHotWater"`
	TimeProgramCirculationPump    TimeProgram `json:"timeProgramCirculationPump"`
}
