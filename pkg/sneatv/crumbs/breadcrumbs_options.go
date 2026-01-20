package crumbs

func WithSeparator(separator string) func(bc *Breadcrumbs) {
	return func(bc *Breadcrumbs) {
		bc.separator = separator
	}
}

func WithSeparatorStartIndex(i int) func(bc *Breadcrumbs) {
	return func(bc *Breadcrumbs) {
		bc.separatorStartIdx = i
	}
}
