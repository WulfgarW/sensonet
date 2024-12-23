package sensonet

func GetDhwData(state SystemStatus, index int) *DhwData {
	// Extracting correct State.Dhw element
	if len(state.State.Dhw) == 0 {
		return nil
	}
	var dhwData DhwData
	for _, stateDhw := range state.State.Dhw {
		if stateDhw.Index == index || (stateDhw.Index == HOTWATERINDEX_DEFAULT && index < 0) {
			dhwData.State = stateDhw
			break
		}
	}
	for _, propDhw := range state.Properties.Dhw {
		if propDhw.Index == index || (propDhw.Index == HOTWATERINDEX_DEFAULT && index < 0) {
			dhwData.Properties = propDhw
			break
		}
	}
	for _, confDhw := range state.Configuration.Dhw {
		if confDhw.Index == index || (confDhw.Index == HOTWATERINDEX_DEFAULT && index < 0) {
			dhwData.Configuration = confDhw
			break
		}
	}
	return &dhwData
}

func GetDomesticHotWaterData(state SystemStatus, index int) *DomesticHotWaterData {
	// Extracting correct State.DomesticHotWater element
	if len(state.State.DomesticHotWater) == 0 {
		return nil
	}
	var domesticHotWaterData DomesticHotWaterData
	for _, stateDomesticHotWater := range state.State.DomesticHotWater {
		if stateDomesticHotWater.Index == index || (stateDomesticHotWater.Index == HOTWATERINDEX_DEFAULT && index < 0) {
			domesticHotWaterData.State = stateDomesticHotWater
			break
		}
	}
	for _, propDomesticHotWater := range state.Properties.DomesticHotWater {
		if propDomesticHotWater.Index == index || (propDomesticHotWater.Index == HOTWATERINDEX_DEFAULT && index < 0) {
			domesticHotWaterData.Properties = propDomesticHotWater
			break
		}
	}
	for _, confDomesticHotWater := range state.Configuration.DomesticHotWater {
		if confDomesticHotWater.Index == index || (confDomesticHotWater.Index == HOTWATERINDEX_DEFAULT && index < 0) {
			domesticHotWaterData.Configuration = confDomesticHotWater
			break
		}
	}
	return &domesticHotWaterData
}

func GetZoneData(state SystemStatus, index int) *ZoneData {
	// Extracting correct State.Zones element
	if len(state.State.Zones) == 0 {
		return nil
	}
	var zoneData ZoneData
	for _, stateZone := range state.State.Zones {
		if stateZone.Index == index || (stateZone.Index == ZONEINDEX_DEFAULT && index < 0) {
			zoneData.State = stateZone
			break
		}
	}
	for _, propZone := range state.Properties.Zones {
		if propZone.Index == index || (propZone.Index == ZONEINDEX_DEFAULT && index < 0) {
			zoneData.Properties = propZone
			break
		}
	}
	for _, confZone := range state.Configuration.Zones {
		if confZone.Index == index || (confZone.Index == ZONEINDEX_DEFAULT && index < 0) {
			zoneData.Configuration = confZone
			break
		}
	}
	return &zoneData
}
