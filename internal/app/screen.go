package app

// activeScreen identifies which screen is currently displayed.
type activeScreen int

const (
	screenReviews  activeScreen = iota
	screenIssues
	screenWatchlist
)
