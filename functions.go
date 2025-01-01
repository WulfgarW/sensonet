package sensonet

func GetDhwData(state *System, index int) *DhwData {
	// Extracting correct State.Dhw element
	if len(state.State.Dhw) == 0 {
		return nil
	}
	var dhwData DhwData
	for _, el := range state.State.Dhw {
		if el.Index == index || (el.Index == HOTWATERINDEX_DEFAULT && index < 0) {
			dhwData.State = el
			break
		}
	}
	for _, el := range state.Properties.Dhw {
		if el.Index == index || (el.Index == HOTWATERINDEX_DEFAULT && index < 0) {
			dhwData.Properties = el
			break
		}
	}
	for _, el := range state.Configuration.Dhw {
		if el.Index == index || (el.Index == HOTWATERINDEX_DEFAULT && index < 0) {
			dhwData.Configuration = el
			break
		}
	}
	return &dhwData
}

func GetDomesticHotWaterData(state *System, index int) *DomesticHotWaterData {
	// Extracting correct State.DomesticHotWater element
	if len(state.State.DomesticHotWater) == 0 {
		return nil
	}
	var domesticHotWaterData DomesticHotWaterData
	for _, el := range state.State.DomesticHotWater {
		if el.Index == index || (el.Index == HOTWATERINDEX_DEFAULT && index < 0) {
			domesticHotWaterData.State = el
			break
		}
	}
	for _, el := range state.Properties.DomesticHotWater {
		if el.Index == index || (el.Index == HOTWATERINDEX_DEFAULT && index < 0) {
			domesticHotWaterData.Properties = el
			break
		}
	}
	for _, el := range state.Configuration.DomesticHotWater {
		if el.Index == index || (el.Index == HOTWATERINDEX_DEFAULT && index < 0) {
			domesticHotWaterData.Configuration = el
			break
		}
	}
	return &domesticHotWaterData
}

func GetZoneData(state *System, index int) *ZoneData {
	// Extracting correct State.Zones element
	if len(state.State.Zones) == 0 {
		return nil
	}
	var zoneData ZoneData
	for _, el := range state.State.Zones {
		if el.Index == index || (el.Index == ZONEINDEX_DEFAULT && index < 0) {
			zoneData.State = el
			break
		}
	}
	for _, el := range state.Properties.Zones {
		if el.Index == index || (el.Index == ZONEINDEX_DEFAULT && index < 0) {
			zoneData.Properties = el
			break
		}
	}
	for _, el := range state.Configuration.Zones {
		if el.Index == index || (el.Index == ZONEINDEX_DEFAULT && index < 0) {
			zoneData.Configuration = el
			break
		}
	}
	return &zoneData
}
