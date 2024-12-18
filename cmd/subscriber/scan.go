package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
)

// I'm going to assume that the message format is fixed and I have no control over it, despite it being in the same repo.
type Scan struct {
	Ip        string   `json:"ip"`
	Port      uint32   `json:"port"`
	Service   string   `json:"service"`
	Timestamp int64    `json:"timestamp"`
	Data      ScanData `json:"data"`
}

type ScanData string

func (s *ScanData) UnmarshalJSON(data []byte) error {
	var dataMap map[string]interface{}
	if err := json.Unmarshal(data, &dataMap); err != nil {
		return err
	}

	if v1data, ok := dataMap["response_bytes_utf8"]; ok {
		dataString, ok := v1data.(string)

		if !ok {
			return fmt.Errorf("error unmarshalling ScanData: expected response_bytes_utf8 to be a string")
		}

		decoded, err := base64.StdEncoding.DecodeString(dataString)
		if err != nil {
			return err
		}
		*s = ScanData(decoded)
		return nil
	}

	if v2data, ok := dataMap["response_str"]; ok {
		dataString, ok := v2data.(string)

		if !ok {
			return fmt.Errorf("error unmarshalling ScanData: expected response_str to be a string")
		}

		*s = ScanData(dataString)
		return nil
	}

	return fmt.Errorf("error unmarshalling ScanData: invalid data format")
}
