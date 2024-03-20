package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
)

// Config struct to hold New Relic API key and alert policy ID
type Config struct {
	APIKey        string `json:"apiKey"`
	AlertPolicyID string `json:"alertPolicyID"`
}

// NRQLCondition struct to represent an NRQL alert condition
type NRQLCondition struct {
	Name    string `json:"name"`
	Terms   []Term `json:"terms"`
	Enabled bool   `json:"enabled"`
}

// Term struct to represent a term within an NRQL alert condition
type Term struct {
	Duration     string `json:"duration"`
	Operator     string `json:"operator"`
	Threshold    string `json:"threshold"`
	TimeFunction string `json:"time_function"`
	Priority     string `json:"priority"`
}

func main() {
	// Load configuration
	config, err := loadConfig("config.json")
	if err != nil {
		fmt.Println("Error loading configuration:", err)
		return
	}

	// Fetch NRQL alert conditions for the specified alert policy ID
	nrqlConditions, err := fetchNRQLConditions(config.APIKey, config.AlertPolicyID)
	if err != nil {
		fmt.Printf("Error fetching NRQL alert conditions: %v\n", err)
		return
	}

	// Print NRQL alert conditions
	fmt.Printf("NRQL Alert Conditions for Policy ID %s:\n", config.AlertPolicyID)
	for _, condition := range nrqlConditions {
		fmt.Printf("Name: %s\n", condition.Name)
		fmt.Println("Terms:")
		for _, term := range condition.Terms {
			fmt.Printf("  Duration: %s\n", term.Duration)
			fmt.Printf("  Operator: %s\n", term.Operator)
			fmt.Printf("  Threshold: %s\n", term.Threshold)
			fmt.Printf("  Time Function: %s\n", term.TimeFunction)
			fmt.Printf("  Priority: %s\n", term.Priority)
		}
		fmt.Println()
	}
	err = SaveNRQLConditionsAsCSV(fmt.Sprintf("%s.csv", config.AlertPolicyID), nrqlConditions)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Println("CSV file saved successfully")
}

// LoadConfig loads configuration from a JSON file
func loadConfig(filename string) (*Config, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	config := &Config{}
	err = decoder.Decode(config)
	if err != nil {
		return nil, err
	}

	return config, nil
}

// FetchNRQLConditions fetches NRQL alert conditions for a specific alert policy from New Relic
func fetchNRQLConditions(apiKey string, policyID string) ([]NRQLCondition, error) {
	url := fmt.Sprintf("https://api.newrelic.com/v2/alerts_nrql_conditions.json?policy_id=%s", policyID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-Api-Key", apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Decode response body
	var data map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	// Extract NRQL alert conditions
	var nrqlConditions []NRQLCondition
	conditions, ok := data["nrql_conditions"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("nrql_conditions field not found or has incorrect type")
	}
	for _, c := range conditions {
		condition, ok := c.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("nrql condition has incorrect type")
		}

		name, ok := condition["name"].(string)
		if !ok {
			return nil, fmt.Errorf("name field is not a string")
		}

		enabled, ok := condition["enabled"].(bool)
		if !ok {
			return nil, fmt.Errorf("enabled field is not a boolean")
		}

		// Extracting terms
		termsData, ok := condition["terms"].([]interface{})
		if !ok {
			return nil, fmt.Errorf("terms field not found or has incorrect type")
		}

		// Iterate over terms
		var terms []Term
		for _, termData := range termsData {
			term, ok := termData.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("term has incorrect type")
			}

			// Extract duration field
			duration, ok := term["duration"].(string)
			if !ok {
				return nil, fmt.Errorf("duration field is not a string")
			}

			operator, ok := term["operator"].(string)
			if !ok {
				return nil, fmt.Errorf("operator field is not a string")
			}

			// Extract and format threshold field
			thresholdValue, ok := term["threshold"].(string)
			if !ok {
				return nil, fmt.Errorf("threshold field is not a string")
			}
			threshold, err := formatThreshold(thresholdValue)
			if err != nil {
				return nil, err
			}

			timeFunction, ok := term["time_function"].(string)
			if !ok {
				return nil, fmt.Errorf("time_function field is not a string")
			}

			priority, ok := term["priority"].(string)
			if !ok {
				return nil, fmt.Errorf("priority field is not a string")
			}

			terms = append(terms, Term{
				Duration:     duration,
				Operator:     operator,
				Threshold:    threshold,
				TimeFunction: timeFunction,
				Priority:     priority,
			})
		}

		nrqlConditions = append(nrqlConditions, NRQLCondition{
			Name:    name,
			Terms:   terms,
			Enabled: enabled,
		})
	}

	return nrqlConditions, nil
}

// formatThreshold formats the threshold value to its exact decimal representation
func formatThreshold(value string) (string, error) {
	f, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%.f", f), nil
}

// SaveNRQLConditionsAsCSV saves NRQL alert conditions to a CSV file
func SaveNRQLConditionsAsCSV(filename string, nrqlConditions []NRQLCondition) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)

	// Write header
	header := []string{"Condition Name", "Duration", "Operator", "Threshold", "Time Function", "Priority", "Active"}
	if err := writer.Write(header); err != nil {
		return err
	}

	// Write data
	for _, condition := range nrqlConditions {
		for _, term := range condition.Terms {
			record := []string{
				condition.Name,
				term.Duration,
				term.Operator,
				term.Threshold,
				term.TimeFunction,
				term.Priority,
				fmt.Sprintf("%t", condition.Enabled),
			}
			if err := writer.Write(record); err != nil {
				return err
			}
		}
	}

	// Flush writer
	writer.Flush()

	if err := writer.Error(); err != nil {
		return err
	}

	return nil
}
