package validation

import (
	"encoding/json"
	"errors"
	"example.com/grain-silo-safety-service/domain"
	"io"
	"net/http"
	"strings"
)

func DecodeInspection(r *http.Request) (domain.InspectionRequest, error) {
	var input domain.InspectionRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 4096)).Decode(&input); err != nil {
		return input, errors.New("body must be valid JSON")
	}
	input.Finding = strings.TrimSpace(input.Finding)
	if input.Finding == "" || len([]rune(input.Finding)) > 300 {
		return input, errors.New("finding must be 1 to 300 characters")
	}
	return input, nil
}
