package masks

import (
	"fmt"
)

type Mask struct {
	Name     string
	Patterns []Pattern
}

func (m *Mask) String() string {
	return fmt.Sprintf("Mask{Name: %q, Patterns: %+v}", m.Name, m.Patterns)
}

func (m *Mask) Match(fileName string) (bool, error) {
	var result bool
	for _, pattern := range m.Patterns {
		matched, err := pattern.Match(fileName)
		if err != nil {
			return false, err
		}
		if matched {
			if pattern.Type == Inclusive {
				result = true
			}
			if pattern.Type == Exclusive {
				return false, nil
			}
		}
	}
	return result, nil
}
