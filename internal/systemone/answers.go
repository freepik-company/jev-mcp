package systemone

import (
	"encoding/json"
	"errors"
	"math"
	"reflect"
	"slices"
	"strconv"
)

const (
	// Explicit allowance for provider rounding; the result itself is never altered.
	roundingTolerance = 0.01
	comparisonEpsilon = 1e-12
)

func validateAnswers(questions map[string]Question, answers map[string]answer) error {
	if len(answers) != len(questions) {
		return errors.New("System One returned an incomplete answer set")
	}
	for id, question := range questions {
		answer, exists := answers[id]
		if !exists || answer.Type != question.Type {
			return errors.New("System One returned a missing or mismatched answer")
		}
		if err := answer.validate(question); err != nil {
			return err
		}
	}
	return nil
}

func (a answer) validate(q Question) error {
	switch q.Type {
	case "noul":
		if !inRange(a.Noul, 0, 1) {
			return errors.New("System One returned an invalid noul probability")
		}
		return nil
	case "choice":
		return a.validateChoice(q.Criteria)
	case "score":
		return a.validateScore(q.Criteria)
	default:
		return errors.New("Unsupported System One question type")
	}
}

func (a answer) validateChoice(raw json.RawMessage) error {
	var criteria map[string]any
	if err := json.Unmarshal(raw, &criteria); err != nil || a.Choice == nil {
		return errors.New("System One returned an invalid choice")
	}
	if _, exists := criteria[*a.Choice]; !exists {
		return errors.New("System One chose an option that was not offered")
	}
	keys := make([]string, 0, len(criteria))
	for key := range criteria {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	if err := a.validateDistribution(keys); err != nil {
		return err
	}
	chosen := *a.Probabilities[*a.Choice]
	for _, probability := range a.Probabilities {
		if chosen+comparisonEpsilon < *probability {
			return errors.New("System One choice contradicts its probability distribution")
		}
	}
	return nil
}

func (a answer) validateScore(raw json.RawMessage) error {
	var criteria []any
	if err := json.Unmarshal(raw, &criteria); err != nil || !inRange(a.Score, 0, float64(len(criteria)-1)) {
		return errors.New("System One returned a score outside the rubric")
	}
	keys := make([]string, len(criteria))
	for index := range criteria {
		keys[index] = strconv.Itoa(index)
	}
	if err := a.validateDistribution(keys); err != nil {
		return err
	}
	var mean float64
	for index, key := range keys {
		mean += float64(index) * *a.Probabilities[key]
	}
	if math.Abs(*a.Score-mean) > roundingTolerance*float64(len(criteria)-1)+comparisonEpsilon {
		return errors.New("System One score contradicts its probability distribution")
	}
	if len(a.Legend) != len(criteria) {
		return errors.New("System One returned an incomplete score legend")
	}
	for index, criterion := range criteria {
		value, exists := a.Legend[keys[index]]
		if !exists || !reflect.DeepEqual(value, criterion) {
			return errors.New("System One score legend differs from the requested rubric")
		}
	}
	return nil
}

func (a answer) validateDistribution(keys []string) error {
	if !inRange(a.Confidence, 0, 1) {
		return errors.New("System One returned invalid confidence")
	}
	if len(keys) != len(a.Probabilities) {
		return errors.New("System One returned an incomplete probability distribution")
	}
	var total float64
	for _, key := range keys {
		probability := a.Probabilities[key]
		if !inRange(probability, 0, 1) {
			return errors.New("System One returned an invalid probability distribution")
		}
		total += *probability
	}
	if math.Abs(total-1) > roundingTolerance+comparisonEpsilon {
		return errors.New("System One returned probabilities that do not sum to one")
	}
	return nil
}

func inRange(value *float64, low, high float64) bool {
	return value != nil && *value >= low && *value <= high
}
