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
