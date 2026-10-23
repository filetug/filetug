package masks

func createBuiltInMasks() []Mask {
	return []Mask{
		{Name: "Coding", Patterns: []Pattern{
			{Type: Inclusive, Regex: `\.(cpp|cs|js|ts|py)$`},
		}},
		{Name: "Data", Patterns: []Pattern{
			{Type: Inclusive, Regex: `\.(csv|dbf|json|xml|yaml)$`},
		}},
	}
}

// BuiltIn returns the masks that ship with FileTug.
func BuiltIn() []Mask { return createBuiltInMasks() }
