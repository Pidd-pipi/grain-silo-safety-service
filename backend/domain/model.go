package domain

type Silo struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Grain          string   `json:"grain"`
	TemperatureC   float64  `json:"temperatureC"`
	MoisturePct    float64  `json:"moisturePct"`
	SafetyState    string   `json:"safetyState"`
	LastInspection string   `json:"lastInspection"`
	Inspected      bool     `json:"inspected"`
	Notes          []string `json:"notes,omitempty"`
}

type InspectionRequest struct {
	Finding string `json:"finding"`
}

